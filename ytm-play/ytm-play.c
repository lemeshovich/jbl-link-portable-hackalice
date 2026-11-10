/* ytm-play — minimal AAC/MP4 over HTTP player for JBL Link Portable.
 *
 * Reads a plain-HTTP URL (the local relay of ytm-alice) or a local file,
 * demuxes MP4 (progressive and fragmented), decodes AAC with faad2 linked
 * statically (the firmware's libfaad_drm.so only accepts DRM streams) and
 * writes PCM to ALSA.
 *
 * libasound.so is loaded with dlopen so that a missing one gives a clear
 * message instead of a loader failure.
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <dlfcn.h>
#include <fcntl.h>
#include <netdb.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/time.h>
#include <signal.h>
#include <pthread.h>

/* ---------- FAAD ---------- */
typedef void *faad_h;
typedef struct {
    unsigned long bytesconsumed;
    unsigned long samples;
    unsigned char channels;
    unsigned char error;
    unsigned long samplerate;
    unsigned char sbr;
    unsigned char object_type;
    unsigned char header_type;
    unsigned char num_front_channels;
    unsigned char num_side_channels;
    unsigned char num_back_channels;
    unsigned char num_lfe_channels;
    unsigned char channel_position[64];
    unsigned char ps;
} faad_frame;

typedef struct {
    unsigned char defObjectType;
    unsigned long defSampleRate;
    unsigned char outputFormat;
    unsigned char downMatrix;
    unsigned char useOldADTSFormat;
    unsigned char dontUpSampleImplicitSBR;
} faad_cfg;

/* linked statically against our own faad2 build (no DRM) */
extern faad_h NeAACDecOpen(void);
extern faad_cfg *NeAACDecGetCurrentConfiguration(faad_h);
extern unsigned char NeAACDecSetConfiguration(faad_h, faad_cfg *);
extern char NeAACDecInit2(faad_h, unsigned char *, unsigned long,
                          unsigned long *, unsigned char *);
extern void *NeAACDecDecode(faad_h, faad_frame *, unsigned char *, unsigned long);
extern void NeAACDecClose(faad_h);
extern char *NeAACDecGetErrorMessage(unsigned char);
#define fa_open   NeAACDecOpen
#define fa_getcfg NeAACDecGetCurrentConfiguration
#define fa_setcfg NeAACDecSetConfiguration
#define fa_init2  NeAACDecInit2
#define fa_decode NeAACDecDecode
#define fa_close  NeAACDecClose

/* ---------- ALSA ---------- */
typedef void snd_pcm_t;
static int (*pcm_open)(snd_pcm_t **, const char *, int, int);
static int (*pcm_set_params)(snd_pcm_t *, int, int, unsigned, unsigned, int, unsigned);
static long (*pcm_writei)(snd_pcm_t *, const void *, unsigned long);
static int (*pcm_recover)(snd_pcm_t *, int, int);
static int (*pcm_drain)(snd_pcm_t *);
static int (*pcm_close)(snd_pcm_t *);

#define SND_PCM_STREAM_PLAYBACK 0
#define SND_PCM_FORMAT_S16_LE   2
#define SND_PCM_ACCESS_RW_INTERLEAVED 3

static int verbose = 0;
static volatile int g_paused = 0;
static volatile int g_stop = 0;
static volatile int g_vol_num = 100;   /* percent, set by "volume <f>" */

/* Reads pause/resume/stop/volume lines from stdin, as ytm-alice expects. */
static void *ctrl_thread(void *arg)
{
    (void)arg;
    char line[128];
    while (fgets(line, sizeof(line), stdin)) {
        char *nl = strchr(line, '\n'); if (nl) *nl = 0;
        if (!strcmp(line, "pause"))       { g_paused = 1; puts("{\"event\":\"paused\"}"); fflush(stdout); }
        else if (!strcmp(line, "resume")) { g_paused = 0; puts("{\"event\":\"playing\"}"); fflush(stdout); }
        else if (!strcmp(line, "stop"))   { g_stop = 1; break; }
        else if (!strncmp(line, "volume ", 7)) {
            double f = atof(line + 7);
            if (f < 0) f = 0; if (f > 1) f = 1;
            g_vol_num = (int)(f * 100 + 0.5);
        }
    }
    return NULL;
}
static snd_pcm_t *g_pcm = NULL;
static void (*g_pcm_close)(void) = NULL;
static int (*pcm_close_p)(snd_pcm_t *) = NULL;
static int (*pcm_drain_p)(snd_pcm_t *) = NULL;
static volatile int g_quit = 0;
static volatile int g_fd = -1;      /* socket/file in use, closed on signal */
static void on_signal(int sig)
{
    (void)sig;
    g_quit = 1;
    /* break the blocking read so the loop can unwind and release dmix */
    if (g_fd >= 0) { shutdown(g_fd, SHUT_RDWR); close(g_fd); g_fd = -1; }
}
static void release_pcm(void)
{
    if (g_pcm && pcm_close_p) { pcm_close_p(g_pcm); g_pcm = NULL; }
}
#define LOG(...) do { if (verbose) fprintf(stderr, __VA_ARGS__); } while (0)

/* ---------- buffered socket reader ---------- */
typedef struct {
    int fd;
    unsigned char buf[65536];
    size_t head, tail;
    int eof;
} reader;

static int rd_fill(reader *r)
{
    if (r->eof || r->fd < 0) return 0;
    if (r->head > 0) {                 /* always compact: keeps room available */
        memmove(r->buf, r->buf + r->head, r->tail - r->head);
        r->tail -= r->head;
        r->head = 0;
    }
    if (r->tail == sizeof(r->buf)) return 0;   /* caller must consume first */
    ssize_t n = read(r->fd, r->buf + r->tail, sizeof(r->buf) - r->tail);
    if (n < 0) { perror("read"); r->eof = 1; return 0; }
    if (n == 0) { r->eof = 1; return 0; }
    r->tail += (size_t)n;
    return 1;
}

/* read exactly n bytes into out (out may be NULL to skip) */
static int rd_exact(reader *r, unsigned char *out, size_t n)
{
    while (n > 0) {
        size_t avail = r->tail - r->head;
        if (avail == 0) {
            if (!rd_fill(r)) return 0;
            continue;
        }
        size_t take = avail < n ? avail : n;
        if (out) { memcpy(out, r->buf + r->head, take); out += take; }
        r->head += take;
        n -= take;
    }
    return 1;
}

static int rd_line(reader *r, char *out, size_t cap)
{
    size_t i = 0;
    for (;;) {
        if (r->head == r->tail && !rd_fill(r)) return 0;
        unsigned char c = r->buf[r->head++];
        if (c == '\n') { out[i] = 0; if (i && out[i-1] == '\r') out[i-1] = 0; return 1; }
        if (i + 1 < cap) out[i++] = (char)c;
    }
}

static unsigned be32(const unsigned char *p)
{ return ((unsigned)p[0] << 24) | (p[1] << 16) | (p[2] << 8) | p[3]; }

/* ---------- HTTP ---------- */
static int http_get(const char *url, reader *r)
{
    char host[256], path[1024];
    int port = 80;
    const char *p = url;
    if (!strncmp(p, "http://", 7)) p += 7;
    else { fprintf(stderr, "only plain http:// is supported\n"); return 0; }
    const char *slash = strchr(p, '/');
    size_t hl = slash ? (size_t)(slash - p) : strlen(p);
    if (hl >= sizeof(host)) return 0;
    memcpy(host, p, hl); host[hl] = 0;
    snprintf(path, sizeof(path), "%s", slash ? slash : "/");
    char *colon = strrchr(host, ':');
    if (colon) { *colon = 0; port = atoi(colon + 1); }

    char portstr[16];
    snprintf(portstr, sizeof(portstr), "%d", port);
    struct addrinfo hints, *res = NULL;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;
    if (getaddrinfo(host, portstr, &hints, &res) != 0 || !res) {
        fprintf(stderr, "cannot resolve %s\n", host); return 0;
    }
    int fd = socket(res->ai_family, res->ai_socktype, 0);
    if (fd < 0 || connect(fd, res->ai_addr, res->ai_addrlen) != 0) {
        fprintf(stderr, "cannot connect to %s:%d\n", host, port);
        if (fd >= 0) close(fd);
        freeaddrinfo(res); return 0;
    }
    freeaddrinfo(res);

    struct timeval tv; tv.tv_sec = 20; tv.tv_usec = 0;
    setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
    setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));

    char req[1600];
    int n = snprintf(req, sizeof(req),
        "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: ytm-play/1\r\n"
        "Accept: */*\r\nRange: bytes=0-\r\nConnection: close\r\n\r\n", path, host);
    if (write(fd, req, (size_t)n) != n) { close(fd); return 0; }

    memset(r, 0, sizeof(*r));
    r->fd = fd;
    fprintf(stderr, "connected to %s:%d, request sent\n", host, port);
    char line[2048];
    if (!rd_line(r, line, sizeof(line))) {
        fprintf(stderr, "no status line (eof=%d)\n", r->eof); close(fd); return 0;
    }
    fprintf(stderr, "status: %s\n", line);
    int code = 0;
    { const char *sp = strchr(line, ' '); if (sp) code = atoi(sp + 1); }
    if (code != 200 && code != 206) {
        fprintf(stderr, "http status %d\n", code); close(fd); return 0;
    }
    int nh = 0;
    while (rd_line(r, line, sizeof(line)) && line[0]) { fprintf(stderr, "hdr: %s\n", line); nh++; }
    fprintf(stderr, "headers done (%d), buffered=%zu\n", nh, r->tail - r->head);
    return 1;
}

/* ---------- MP4 ---------- */
static unsigned char asc[64];
static size_t asc_len = 0;

/* find esds inside stsd payload and copy its DecoderSpecificInfo */
static void scan_for_asc(const unsigned char *b, size_t n)
{
    for (size_t i = 0; i + 8 <= n; i++) {
        if (memcmp(b + i, "esds", 4) != 0) continue;
        size_t j = i + 4 + 4;                 /* skip tag + version/flags */
        while (j < n) {
            unsigned char tag = b[j++];
            unsigned long len = 0;
            for (int k = 0; k < 4 && j < n; k++) {
                unsigned char c = b[j++];
                len = (len << 7) | (c & 0x7f);
                if (!(c & 0x80)) break;
            }
            if (tag == 0x03) { j += 3; continue; }          /* ES_Descr */
            if (tag == 0x04) { j += 13; continue; }         /* DecoderConfig */
            if (tag == 0x05) {                              /* DecoderSpecificInfo */
                if (len > sizeof(asc)) len = sizeof(asc);
                if (j + len <= n) { memcpy(asc, b + j, len); asc_len = len; }
                return;
            }
            j += len;
        }
        return;
    }
}

/* sample sizes collected from the current traf/trun or stsz */
static unsigned *sizes = NULL;
static size_t sizes_n = 0, sizes_cap = 0;

static void sizes_reset(void) { sizes_n = 0; }
static void sizes_push(unsigned v)
{
    if (sizes_n == sizes_cap) {
        size_t cap = sizes_cap ? sizes_cap * 2 : 1024;
        unsigned *p = realloc(sizes, cap * sizeof(*p));
        if (!p) return;
        sizes = p; sizes_cap = cap;
    }
    sizes[sizes_n++] = v;
}

static void parse_trun(const unsigned char *b, size_t n)
{
    if (n < 8) return;
    unsigned flags = be32(b) & 0xffffff;
    unsigned count = be32(b + 4);
    size_t off = 8;
    if (flags & 0x000001) off += 4;   /* data offset */
    if (flags & 0x000004) off += 4;   /* first sample flags */
    unsigned per = 0;
    if (flags & 0x000100) per += 4;   /* duration */
    if (flags & 0x000200) per += 4;   /* size */
    if (flags & 0x000400) per += 4;   /* flags */
    if (flags & 0x000800) per += 4;   /* cts offset */
    if (!(flags & 0x000200)) return;  /* no per-sample size: unsupported */
    for (unsigned i = 0; i < count && off + per <= n; i++) {
        size_t q = off;
        if (flags & 0x000100) q += 4;
        sizes_push(be32(b + q));
        off += per;
    }
}

static void parse_stsz(const unsigned char *b, size_t n)
{
    if (n < 12) return;
    unsigned uniform = be32(b + 4);
    unsigned count = be32(b + 8);
    if (uniform) { for (unsigned i = 0; i < count; i++) sizes_push(uniform); return; }
    for (unsigned i = 0; i < count && 12 + 4 * (size_t)i + 4 <= n; i++)
        sizes_push(be32(b + 12 + 4 * (size_t)i));
}

/* recursively walk a fully buffered container box */
static void walk(const unsigned char *b, size_t n)
{
    size_t off = 0;
    while (off + 8 <= n) {
        unsigned sz = be32(b + off);
        char type[5]; memcpy(type, b + off + 4, 4); type[4] = 0;
        size_t hdr = 8;
        if (sz == 1) break;                 /* 64-bit size: not expected here */
        if (sz < 8 || off + sz > n) break;
        const unsigned char *pay = b + off + hdr;
        size_t paylen = sz - hdr;
        if (!strcmp(type, "trak") || !strcmp(type, "mdia") ||
            !strcmp(type, "minf") || !strcmp(type, "stbl") ||
            !strcmp(type, "traf") || !strcmp(type, "moof") ||
            !strcmp(type, "mvex"))
            walk(pay, paylen);
        else if (!strcmp(type, "stsd")) scan_for_asc(pay, paylen);
        else if (!strcmp(type, "stsz")) parse_stsz(pay, paylen);
        else if (!strcmp(type, "trun")) parse_trun(pay, paylen);
        off += sz;
    }
}

int main(int argc, char **argv)
{
    const char *url = NULL, *dev = "default";
    int gain_pct = 100;             /* software volume, 0..100 */
    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-v")) verbose = 1;
        else if (!strcmp(argv[i], "-D") && i + 1 < argc) dev = argv[++i];
        else if (!strcmp(argv[i], "-g") && i + 1 < argc) {
            gain_pct = atoi(argv[++i]);
            if (gain_pct < 0) gain_pct = 0;
            if (gain_pct > 100) gain_pct = 100;
        }
        else url = argv[i];
    }
    if (!url) {
        fprintf(stderr, "usage: ytm-play [-v] [-g 0..100] [-D alsa-device] <url|file>\n");
        return 2;
    }

    void *la = dlopen("libasound.so", RTLD_NOW);
    if (!la) la = dlopen("libasound.so.2", RTLD_NOW);
    if (!la) { fprintf(stderr, "libasound.so: %s\n", dlerror()); return 1; }

    pcm_open  = dlsym(la, "snd_pcm_open");
    pcm_set_params = dlsym(la, "snd_pcm_set_params");
    pcm_writei = dlsym(la, "snd_pcm_writei");
    pcm_recover = dlsym(la, "snd_pcm_recover");
    pcm_drain = dlsym(la, "snd_pcm_drain");
    pcm_close = dlsym(la, "snd_pcm_close");
    if (!pcm_open || !pcm_set_params || !pcm_writei) {
        fprintf(stderr, "required symbols missing\n"); return 1;
    }

    {
        pthread_t th;
        if (pthread_create(&th, NULL, ctrl_thread, NULL) == 0) pthread_detach(th);
    }
    signal(SIGINT, on_signal); signal(SIGTERM, on_signal); signal(SIGHUP, on_signal);
    signal(SIGPIPE, SIG_IGN);
    pcm_close_p = pcm_close; pcm_drain_p = pcm_drain;
    atexit(release_pcm);

    reader r;
    if (!strncmp(url, "http://", 7)) {
        if (!http_get(url, &r)) return 1;
        g_fd = r.fd;
    } else {
        memset(&r, 0, sizeof(r));
        r.fd = open(url, O_RDONLY);
        if (r.fd < 0) { fprintf(stderr, "cannot open %s\n", url); return 1; }
        g_fd = r.fd;
        LOG("reading local file %s\n", url);
    }

    faad_h dec = fa_open();
    if (!dec) { fprintf(stderr, "NeAACDecOpen failed\n"); return 1; }
    {
        faad_cfg *c = fa_getcfg(dec);
        if (c) { c->outputFormat = 1; /* 16-bit */ fa_setcfg(dec, c); }
    }

    snd_pcm_t *pcm = NULL;
    int opened = 0;
    unsigned long rate = 0; unsigned char chans = 0;
    unsigned char *box = NULL; size_t box_cap = 0;
    long long total_frames = 0, last_report = 0;

    while (!g_quit) {
        unsigned char hdr[8];
        if (!rd_exact(&r, hdr, 8)) {
            fprintf(stderr, "stopped: cannot read box header (eof=%d, buffered=%zu)\n",
                    r.eof, r.tail - r.head);
            break;
        }
        unsigned sz = be32(hdr);
        char type[5]; memcpy(type, hdr + 4, 4); type[4] = 0;
        if (sz < 8) { LOG("bad box size %u (%s)\n", sz, type); break; }
        size_t paylen = sz - 8;
        fprintf(stderr, "box %s size %u\n", type, sz);

        if (!strcmp(type, "mdat")) {
            if (asc_len == 0) { fprintf(stderr, "mdat before config\n"); break; }
            if (!opened) {
                if (fa_init2(dec, asc, asc_len, &rate, &chans) < 0) {
                    fprintf(stderr, "NeAACDecInit2 failed\n"); break;
                }
                fprintf(stderr, "aac: %lu Hz, %u ch\n", rate, chans);
                if (pcm_open(&pcm, dev, SND_PCM_STREAM_PLAYBACK, 0) < 0) {
                    fprintf(stderr, "cannot open alsa device %s\n", dev); break;
                }
                if (pcm_set_params(pcm, SND_PCM_FORMAT_S16_LE,
                                   SND_PCM_ACCESS_RW_INTERLEAVED,
                                   chans ? chans : 2, (unsigned)(rate ? rate : 44100),
                                   1, 300000) < 0) {
                    fprintf(stderr, "snd_pcm_set_params failed\n"); break;
                }
                opened = 1; g_pcm = pcm;
                puts("{\"event\":\"playing\"}"); fflush(stdout);
            }
            size_t left = paylen, idx = 0;
            unsigned char *frame = malloc(1 << 16);
            if (!frame) break;
            while (left > 0 && !g_quit) {
                size_t want;
                if (idx < sizes_n) want = sizes[idx++];
                else want = left;                 /* no table: take the rest */
                if (want == 0 || want > left) want = left;
                if (want > (1 << 16)) want = 1 << 16;
                if (!rd_exact(&r, frame, want)) { left = 0; break; }
                left -= want;
                faad_frame fi;
                memset(&fi, 0, sizeof(fi));
                void *pcmbuf = fa_decode(dec, &fi, frame, want);
                if (fi.error) { LOG("decode error %u: %s\n", fi.error, NeAACDecGetErrorMessage(fi.error)); continue; }
                if (!pcmbuf || fi.samples == 0) continue;
                unsigned long frames = fi.samples / (fi.channels ? fi.channels : 1);
                while (g_paused && !g_quit && !g_stop) usleep(100000);
                if (g_stop) { g_quit = 1; }
                int vol = gain_pct * g_vol_num / 100;
                if (vol != 100) {
                    short *q = pcmbuf;
                    for (unsigned long k = 0; k < fi.samples; k++)
                        q[k] = (short)((int)q[k] * vol / 100);
                }
                const short *s = pcmbuf;
                while (frames > 0) {
                    long w = pcm_writei(pcm, s, frames);
                    if (w < 0) {
                        if (pcm_recover && pcm_recover(pcm, (int)w, 1) == 0) continue;
                        LOG("write error %ld\n", w); frames = 0; break;
                    }
                    s += (size_t)w * (fi.channels ? fi.channels : 1);
                    frames -= (unsigned long)w;
                    total_frames += w;
                    if (rate && total_frames - last_report >= (long long)rate) {
                        last_report = total_frames;
                        printf("{\"event\":\"position\",\"position_ms\":%lld}\n",
                               total_frames * 1000 / (long long)rate);
                        fflush(stdout);
                    }
                }
            }
            free(frame);
            sizes_reset();
            continue;
        }

        /* buffer the whole box and walk it */
        if (paylen > 16u << 20) { rd_exact(&r, NULL, paylen); continue; }
        if (paylen > box_cap) {
            unsigned char *p = realloc(box, paylen);
            if (!p) break;
            box = p; box_cap = paylen;
        }
        if (!rd_exact(&r, box, paylen)) break;
        if (!strcmp(type, "moov") || !strcmp(type, "moof")) walk(box, paylen);
    }

    if (opened && pcm) {
        if (!g_quit && pcm_drain) pcm_drain(pcm);
        if (pcm_close) pcm_close(pcm);
        g_pcm = NULL;
    }
    fa_close(dec);
    if (r.fd >= 0) close(r.fd);
    g_fd = -1;
    free(box); free(sizes);
    if (total_frames > 0 && !g_stop && !g_quit) { puts("{\"event\":\"eos\"}"); fflush(stdout); }
    if (total_frames == 0)
        { puts("{\"event\":\"error\"}"); fflush(stdout); }
    if (total_frames == 0)
        fprintf(stderr, "ytm-play: no audio decoded (asc=%zu bytes, eof=%d)\n", asc_len, r.eof);
    else
        LOG("played %lld frames\n", total_frames);
    return total_frames > 0 ? 0 : 1;
}
