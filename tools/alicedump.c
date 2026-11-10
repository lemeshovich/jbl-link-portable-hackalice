/* alicedump — connect to quasar's aliced IPC (127.0.0.1:9875) and print the
 * structure of every frame, so the field that carries the recognised phrase
 * can be identified on this firmware.
 *
 * Framing: 4-byte little-endian length, then a protobuf message.
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <ctype.h>
#include <signal.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

static int printable(const unsigned char *b, size_t n)
{
    if (n == 0 || n > 400) return 0;
    size_t ok = 0;
    for (size_t i = 0; i < n; i++)
        if (b[i] >= 0x20 || b[i] == '\n' || b[i] == '\t' || b[i] >= 0xc0 || (b[i] & 0x80)) ok++;
    return ok * 10 >= n * 9;
}

static size_t rv(const unsigned char *b, size_t n, size_t *i, unsigned long long *out)
{
    unsigned long long v = 0; int sh = 0;
    while (*i < n) {
        unsigned char c = b[(*i)++];
        v |= (unsigned long long)(c & 0x7f) << sh; sh += 7;
        if (!(c & 0x80)) { *out = v; return 1; }
        if (sh > 63) break;
    }
    return 0;
}

static void dump(const unsigned char *b, size_t n, int depth, const char *path)
{
    size_t i = 0;
    while (i < n) {
        unsigned long long key;
        if (!rv(b, n, &i, &key)) return;
        int field = (int)(key >> 3), wire = (int)(key & 7);
        char np[128];
        snprintf(np, sizeof(np), "%s%s%d", path, path[0] ? "." : "", field);
        if (wire == 0) {
            unsigned long long v;
            if (!rv(b, n, &i, &v)) return;
            printf("%*s[%s] = %llu\n", depth * 2, "", np, v);
        } else if (wire == 2) {
            unsigned long long len;
            if (!rv(b, n, &i, &len) || i + len > n) return;
            const unsigned char *p = b + i;
            int looks_nested = 0;
            if (len >= 2) {
                size_t j = 0; unsigned long long k2;
                if (rv(p, len, &j, &k2)) {
                    int f2 = (int)(k2 >> 3), w2 = (int)(k2 & 7);
                    if (f2 > 0 && f2 < 512 && (w2 == 0 || w2 == 2)) looks_nested = 1;
                }
            }
            if (printable(p, len) && !looks_nested) {
                printf("%*s[%s] = \"%.*s\"\n", depth * 2, "", np, (int)len, p);
            } else {
                printf("%*s[%s] { %llu bytes\n", depth * 2, "", np, len);
                if (depth < 5) dump(p, len, depth + 1, np);
                printf("%*s}\n", depth * 2, "");
            }
            i += len;
        } else if (wire == 5) { i += 4; }
        else if (wire == 1) { i += 8; }
        else return;
    }
}

static volatile int quit = 0;
static void onsig(int s) { (void)s; quit = 1; }

int main(int argc, char **argv)
{
    int port = argc > 1 ? atoi(argv[1]) : 9875;
    signal(SIGINT, onsig); signal(SIGTERM, onsig);

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in a;
    memset(&a, 0, sizeof(a));
    a.sin_family = AF_INET;
    a.sin_port = htons((unsigned short)port);
    a.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (fd < 0 || connect(fd, (struct sockaddr *)&a, sizeof(a)) != 0) {
        perror("connect"); return 1;
    }
    fprintf(stderr, "listening to aliced on 127.0.0.1:%d — speak to the speaker, Ctrl+C to stop\n", port);

    unsigned char hdr[4], *buf = malloc(1 << 20);
    if (!buf) return 1;
    int frame = 0;
    while (!quit) {
        size_t got = 0;
        while (got < 4) {
            ssize_t r = read(fd, hdr + got, 4 - got);
            if (r <= 0) { free(buf); close(fd); return 0; }
            got += (size_t)r;
        }
        unsigned len = hdr[0] | (hdr[1] << 8) | (hdr[2] << 16) | ((unsigned)hdr[3] << 24);
        if (len == 0 || len > (1u << 20)) { fprintf(stderr, "bad frame length %u\n", len); break; }
        got = 0;
        while (got < len) {
            ssize_t r = read(fd, buf + got, len - got);
            if (r <= 0) { free(buf); close(fd); return 0; }
            got += (size_t)r;
        }
        printf("--- frame %d, %u bytes ---\n", ++frame, len);
        dump(buf, len, 0, "");
        fflush(stdout);
    }
    free(buf); close(fd);
    return 0;
}
