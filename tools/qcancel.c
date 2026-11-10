/* qcancel — cancel Alice's current dialog on quasar 3.21.
 *
 * Sends QuasarMessage { restart_vins_dialog(86) = { need_cancel(1) = true } }
 * to aliced on 127.0.0.1:9875, using the 4-byte little-endian length framing.
 *
 * Field numbers were recovered from the protobuf descriptors embedded in the
 * firmware's own maind binary, so they match this build exactly.
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

static size_t pv(unsigned char *b, size_t n, unsigned long long v)
{ do { unsigned char c = v & 0x7f; v >>= 7; if (v) c |= 0x80; b[n++] = c; } while (v); return n; }
static size_t pb(unsigned char *b, size_t n, int f, const unsigned char *v, size_t l)
{ n = pv(b, n, ((unsigned long long)f << 3) | 2); n = pv(b, n, l); memcpy(b + n, v, l); return n + l; }
static size_t pi(unsigned char *b, size_t n, int f, unsigned long long v)
{ n = pv(b, n, ((unsigned long long)f << 3) | 0); return pv(b, n, v); }

int main(int argc, char **argv)
{
    int field = 86, port = 9875, val = 1, sub = -1, empty = 0;
    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-f") && i + 1 < argc) field = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-p") && i + 1 < argc) port = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-v") && i + 1 < argc) val = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-s") && i + 1 < argc) { sub = atoi(argv[++i]); }
        else if (!strcmp(argv[i], "-e")) empty = 1;
        else if (!strcmp(argv[i], "-E")) sub = -2;   /* top-level empty message */
        else { fprintf(stderr,
            "usage: qcancel [-f field] [-s subfield] [-e] [-v value] [-p port]\n"
            "  -e  make the subfield an empty nested message\n"); return 2; }
    }
    unsigned char inner[32]; size_t il = 0;
    if (sub == -2) {
        il = 0;                                  /* EmptyMessage payload */
    } else if (sub >= 0) {
        /* field { sub { } } or field { sub = val } */
        if (empty) il = pb(inner, 0, sub, (const unsigned char *)"", 0);
        else       il = pi(inner, 0, sub, (unsigned)val);
    } else {
        il = pi(inner, 0, 1, (unsigned)val);
    }
    unsigned char msg[128]; size_t ml = pb(msg, 0, field, inner, il);

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in a; memset(&a, 0, sizeof(a));
    a.sin_family = AF_INET; a.sin_port = htons((unsigned short)port);
    a.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (fd < 0 || connect(fd, (struct sockaddr *)&a, sizeof(a)) != 0) { perror("connect"); return 1; }
    unsigned char h[4];
    h[0] = ml & 0xff; h[1] = (ml >> 8) & 0xff; h[2] = (ml >> 16) & 0xff; h[3] = (ml >> 24) & 0xff;
    if (write(fd, h, 4) != 4 || write(fd, msg, ml) != (ssize_t)ml) { perror("write"); close(fd); return 1; }
    fprintf(stderr, "sent field=%d sub=%d empty=%d value=%d (%zu bytes)\n",
            field, sub, empty, val, ml);
    usleep(300000);
    close(fd);
    return 0;
}
