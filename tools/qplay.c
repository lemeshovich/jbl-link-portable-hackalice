/* qplay — ask quasar's audioclient (127.0.0.1:9906) to play a URL.
 *
 * Builds QuasarMessage { media_request(8) = MediaRequest { play_audio(1) =
 * Audio { id(1), url(2), format(4) } } } and writes it with the quasar IPC
 * framing: 4-byte little-endian length, then the protobuf body.
 *
 * Playback is done by the firmware itself (souphttpsrc ! decodebin !
 * alsasink), so the DSP stays configured the way the firmware expects.
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <sys/time.h>

static unsigned char out[8192];
static size_t olen = 0;

static void put_varint(unsigned char *b, size_t *n, unsigned long long v)
{
    do { unsigned char c = v & 0x7f; v >>= 7; if (v) c |= 0x80; b[(*n)++] = c; } while (v);
}

/* append key+payload (wire type 2) */
static size_t emit_bytes(unsigned char *b, size_t n, int field,
                         const unsigned char *val, size_t vlen)
{
    put_varint(b, &n, ((unsigned long long)field << 3) | 2);
    put_varint(b, &n, vlen);
    memcpy(b + n, val, vlen); n += vlen;
    return n;
}

static size_t emit_varint_field(unsigned char *b, size_t n, int field, unsigned long long v)
{
    put_varint(b, &n, ((unsigned long long)field << 3) | 0);
    put_varint(b, &n, v);
    return n;
}

int main(int argc, char **argv)
{
    const char *url = NULL, *id = "ytm";
    int do_stop = 0;
    int format = 1;            /* Audio.Format: 1 = MP3 (decodebin path) */
    int port = 9906, verbose = 0;
    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-v")) verbose = 1;
        else if (!strcmp(argv[i], "-f") && i + 1 < argc) format = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-p") && i + 1 < argc) port = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-i") && i + 1 < argc) id = argv[++i];
        else if (!strcmp(argv[i], "-stop")) do_stop = 1;
        else url = argv[i];
    }
    if (!url && !do_stop) {
        fprintf(stderr, "usage: qplay [-v] [-f format] [-p port] [-i id] <url>\n"
                        "       qplay -stop     release players / audio focus\n"
                        "  format: 1=MP3/decodebin 3=HLS 4=AUDIO_FILE 5=OPUS\n");
        return 2;
    }

    if (do_stop) {
        /* MediaRequest { clean_players(31) = CleanPlayers{} } */
        unsigned char empty[4]; size_t elen = 0;
        unsigned char mr[64]; size_t ml = 0;
        ml = emit_bytes(mr, ml, 31, empty, elen);
        olen = emit_bytes(out, olen, 8, mr, ml);
        goto send;
    }

    /* Audio { id=1, url=2, format=4 } */
    unsigned char audio[4096]; size_t alen = 0;
    alen = emit_bytes(audio, alen, 1, (const unsigned char *)id, strlen(id));
    alen = emit_bytes(audio, alen, 2, (const unsigned char *)url, strlen(url));
    alen = emit_varint_field(audio, alen, 4, (unsigned)format);

    /* MediaRequest { play_audio=1 } */
    unsigned char mreq[5120]; size_t mlen = 0;
    mlen = emit_bytes(mreq, mlen, 1, audio, alen);

    /* QuasarMessage { media_request=8 } */
    olen = emit_bytes(out, olen, 8, mreq, mlen);

send:
    if (verbose) {
        fprintf(stderr, "payload %zu bytes:", olen);
        for (size_t i = 0; i < olen && i < 64; i++) fprintf(stderr, " %02x", out[i]);
        fprintf(stderr, "%s\n", olen > 64 ? " ..." : "");
    }

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in a;
    memset(&a, 0, sizeof(a));
    a.sin_family = AF_INET;
    a.sin_port = htons((unsigned short)port);
    a.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (fd < 0 || connect(fd, (struct sockaddr *)&a, sizeof(a)) != 0) {
        perror("connect"); return 1;
    }
    struct timeval tv = { 5, 0 };
    setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

    unsigned char hdr[4];
    hdr[0] = olen & 0xff; hdr[1] = (olen >> 8) & 0xff;
    hdr[2] = (olen >> 16) & 0xff; hdr[3] = (olen >> 24) & 0xff;
    if (write(fd, hdr, 4) != 4 || write(fd, out, olen) != (ssize_t)olen) {
        perror("write"); close(fd); return 1;
    }
    fprintf(stderr, "sent %zu bytes to 127.0.0.1:%d\n", olen, port);

    unsigned char rsp[512];
    ssize_t n = read(fd, rsp, sizeof(rsp));
    if (n > 0) {
        fprintf(stderr, "reply %zd bytes:", n);
        for (ssize_t i = 0; i < n && i < 48; i++) fprintf(stderr, " %02x", rsp[i]);
        fprintf(stderr, "\n");
    } else {
        fprintf(stderr, "no reply (this may still be fine)\n");
    }
    close(fd);
    return 0;
}
