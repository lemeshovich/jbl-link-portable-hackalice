/* ipcclean — list and remove System V shm/sem segments by key.
 * BusyBox has no ipcs/ipcrm, and a stale dmix segment (ipc_key 1024)
 * blocks all audio until it is removed.
 */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ipc.h>
#include <sys/shm.h>
#include <sys/sem.h>

int main(int argc, char **argv)
{
    if (argc < 2) {
        fprintf(stderr, "usage: ipcclean <key> [<key> ...]   remove shm+sem by key\n"
                        "       ipcclean dmix                shorthand for 1024 1025\n");
        return 2;
    }
    int keys[16], nk = 0;
    for (int i = 1; i < argc && nk < 16; i++) {
        if (!strcmp(argv[i], "dmix")) { keys[nk++] = 1024; keys[nk++] = 1025; }
        else keys[nk++] = (int)strtol(argv[i], NULL, 0);
    }
    for (int i = 0; i < nk; i++) {
        int id = shmget(keys[i], 0, 0);
        if (id >= 0) {
            struct shmid_ds ds;
            if (shmctl(id, IPC_STAT, &ds) == 0)
                printf("shm key=%d id=%d nattch=%lu size=%zu\n",
                       keys[i], id, (unsigned long)ds.shm_nattch, (size_t)ds.shm_segsz);
            if (shmctl(id, IPC_RMID, NULL) == 0) printf("  shm removed\n");
            else perror("  shmctl IPC_RMID");
        } else printf("shm key=%d: not present\n", keys[i]);

        int sid = semget(keys[i], 0, 0);
        if (sid >= 0) {
            if (semctl(sid, 0, IPC_RMID) == 0) printf("  sem key=%d removed\n", keys[i]);
            else perror("  semctl IPC_RMID");
        }
    }
    puts("done");
    return 0;
}
