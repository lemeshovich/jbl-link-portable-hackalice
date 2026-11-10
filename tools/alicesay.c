/* alicesay — make Alice speak a phrase on quasar 3.21 (jbl_link_portable).
 *
 * Default: QuasarMessage.vins_server_action(49) = VinsServerAction {
 *   server_action(1) = <update_form JSON for repeat_after_me> }
 * This is the same server action the YandexStation Home Assistant
 * integration sends over glagol to make the speaker say arbitrary text.
 *
 * -c  fallback: QuasarMessage.vins_call(79) = VinsCall { text_input(1) }
 *
 * Field numbers come from the descriptors embedded in this firmware's maind.
 * Framing: 4-byte little-endian length, then the protobuf body.
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
static size_t pb(unsigned char *b, size_t n, int f, const void *v, size_t l)
{ n = pv(b, n, ((unsigned long long)f << 3) | 2); n = pv(b, n, l); memcpy(b + n, v, l); return n + l; }
static size_t pi(unsigned char *b, size_t n, int f, unsigned long long v)
{ n = pv(b, n, ((unsigned long long)f << 3) | 0); return pv(b, n, v); }

int main(int argc, char **argv)
{
    const char *text = "\xd0\x92\xd0\xba\xd0\xbb\xd1\x8e\xd1\x87\xd0\xb0\xd1\x8e"; /* "Включаю" */
    int use_call = 0, port = 9875;
    int silent = -1, reset = -1, interaction = -1;   /* -1 = field omitted */
    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-c")) use_call = 1;
        else if (!strcmp(argv[i], "-p") && i + 1 < argc) port = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-s") && i + 1 < argc) silent = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-r") && i + 1 < argc) reset = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-i") && i + 1 < argc) interaction = atoi(argv[++i]);
        else text = argv[i];
    }

    static unsigned char inner[4096], msg[4200];
    size_t il, ml;
    if (use_call) {
        /* "повтори за мной " + text */
        char q[1024];
        snprintf(q, sizeof(q), "%s %s",
                 "\xd0\xbf\xd0\xbe\xd0\xb2\xd1\x82\xd0\xbe\xd1\x80\xd0\xb8 \xd0\xb7\xd0\xb0 \xd0\xbc\xd0\xbd\xd0\xbe\xd0\xb9", text);
        il = pb(inner, 0, 1, q, strlen(q));
        ml = pb(msg, 0, 79, inner, il);
    } else {
        char js[2048];
        snprintf(js, sizeof(js),
            "{\"type\":\"server_action\",\"name\":\"update_form\",\"payload\":{"
            "\"form_update\":{\"name\":\"personal_assistant.scenarios.repeat_after_me\","
            "\"slots\":[{\"type\":\"string\",\"name\":\"request\",\"value\":\"%s\"}]},"
            "\"resubmit\":true}}", text);
        il = pb(inner, 0, 1, js, strlen(js));
        /* VinsServerAction: reset_session=2, has_interaction=3, is_silent=5 */
        if (reset >= 0)       il = pi(inner, il, 2, (unsigned)reset);
        if (interaction >= 0) il = pi(inner, il, 3, (unsigned)interaction);
        if (silent >= 0)      il = pi(inner, il, 5, (unsigned)silent);
        ml = pb(msg, 0, 49, inner, il);
    }

    int fd = socket(AF_INET, SOCK_STREAM, 0);
    struct sockaddr_in a; memset(&a, 0, sizeof(a));
    a.sin_family = AF_INET; a.sin_port = htons((unsigned short)port);
    a.sin_addr.s_addr = inet_addr("127.0.0.1");
    if (fd < 0 || connect(fd, (struct sockaddr *)&a, sizeof(a)) != 0) { perror("connect"); return 1; }
    unsigned char h[4] = { ml & 0xff, (ml >> 8) & 0xff, (ml >> 16) & 0xff, (ml >> 24) & 0xff };
    if (write(fd, h, 4) != 4 || write(fd, msg, ml) != (ssize_t)ml) { perror("write"); close(fd); return 1; }
    fprintf(stderr, "sent %s silent=%d reset=%d interaction=%d (%zu bytes)\n",
            use_call ? "vins_call(79)" : "vins_server_action(49)", silent, reset, interaction, ml);
    usleep(300000);
    close(fd);
    return 0;
}
