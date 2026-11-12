package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Alice IPC: aliced listens on 127.0.0.1:9875. Every message is a 4-byte
// little-endian length followed by a protobuf QuasarMessage. Field numbers
// below are those of quasar 3.21 (JBL Link Portable, firmware 2021).

const ipcAddr = "127.0.0.1:9875"

// ---- minimal protobuf ----

type field struct {
	wire int
	n    uint64
	b    []byte
}

func uvarint(b []byte, i int) (uint64, int, bool) {
	var v uint64
	for s := uint(0); i < len(b) && s < 64; s += 7 {
		c := b[i]
		i++
		v |= uint64(c&0x7f) << s
		if c < 0x80 {
			return v, i, true
		}
	}
	return 0, i, false
}

// parse returns the last occurrence of every top-level field.
func parse(b []byte) map[int]field {
	m := map[int]field{}
	for i := 0; i < len(b); {
		key, j, ok := uvarint(b, i)
		if !ok {
			break
		}
		i = j
		f := field{wire: int(key & 7)}
		switch f.wire {
		case 0:
			v, j, ok := uvarint(b, i)
			if !ok {
				return m
			}
			f.n, i = v, j
		case 2:
			l, j, ok := uvarint(b, i)
			if !ok || j+int(l) > len(b) {
				return m
			}
			f.b, i = b[j:j+int(l)], j+int(l)
		case 1:
			i += 8
		case 5:
			i += 4
		default:
			return m
		}
		m[int(key>>3)] = f
	}
	return m
}

func putVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pb(f int, payload []byte) []byte {
	b := putVarint(nil, uint64(f)<<3|2)
	b = putVarint(b, uint64(len(payload)))
	return append(b, payload...)
}

func pi(f int, v uint64) []byte {
	return putVarint(putVarint(nil, uint64(f)<<3), v)
}

// ---- events ----

type event struct {
	HasState bool
	State    uint64 // 0 idle, 1 listening, 2 speaking, 3 busy
	Text     string // recognised phrase
	Speech   string // what Alice is going to say or is saying
	Spotted  bool   // wake word
}

func decode(b []byte) event {
	d := parse(b)
	// QuasarMessage.speech_kit_event (43) .on_phrase_spotted (1)
	spotted := len(parse(d[43].b)[1].b) > 0
	// Newer firmware wraps the state in an AliceCapability envelope.
	if d[253].wire == 0 && d[253].n == 3 {
		d = map[int]field{3: {wire: 2, b: d[252].b}}
	}
	_, has := d[3]
	st := parse(d[3].b)
	e := event{HasState: has, State: st[1].n, Text: string(st[2].b), Spotted: spotted}
	if reply := parse(st[3].b); len(reply[1].b) > 0 {
		e.Speech = string(reply[1].b)
	}
	return e
}

func readFrame(r io.Reader) ([]byte, error) {
	var h [4]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(h[:])
	if n > 4<<20 {
		return nil, errors.New("frame too large")
	}
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return b, err
}

func writeFrame(w io.Writer, msg []byte) error {
	var h [4]byte
	binary.LittleEndian.PutUint32(h[:], uint32(len(msg)))
	_, err := w.Write(append(h[:], msg...))
	return err
}

// send writes one message and keeps the connection open half a second before
// closing it: aliced drops messages from connections that close too early.
func send(msg []byte) error {
	c, err := net.DialTimeout("tcp", ipcAddr, 2*time.Second)
	if err != nil {
		return err
	}
	c.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := writeFrame(c, msg); err != nil {
		c.Close()
		return err
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		c.Close()
	}()
	return nil
}

// cancelDialog is QuasarMessage.cleanup_directive_controller (170): the
// firmware's own dialog cancel. It also cancels a request in flight.
func cancelDialog() error { return send(pb(170, nil)) }

// serverAction sends QuasarMessage.vins_server_action (49) with a form update.
func serverAction(scenario, slot, text string) error {
	v, err := json.Marshal(text)
	if err != nil {
		return err
	}
	action := `{"type":"server_action","name":"update_form","payload":{"form_update":` +
		`{"name":"` + scenario + `","slots":` +
		`[{"type":"string","name":"` + slot + `","value":` + string(v) + `}]},"resubmit":true}}`
	return send(pb(49, pb(1, []byte(action))))
}

// say makes Alice speak text in her own voice through the smart-home phrase
// scenario. Unlike the dialog scenario "repeat after me", this one is not
// filtered by Yandex, takes long texts, and does not open the microphone
// afterwards.
func say(text string) error {
	return serverAction("personal_assistant.scenarios.quasar.iot.repeat_phrase",
		"phrase_to_repeat", text)
}

// sayDialog speaks through the dialog scenario "repeat after me". Unlike say,
// Alice opens a proper listening session once she is done, so the user can
// answer without the wake word. Yandex filters some topics in this scenario.
func sayDialog(text string) error {
	return serverAction("personal_assistant.scenarios.repeat_after_me", "request", text)
}

// ---- Alice's voice ----

// Alice speaks through the "default" PCM, whose softvol is ShadowMaster; music
// players use Master. The saved value lives in the same file ytm-alice uses, so
// the two daemons never save each other's zero. Only control elements are
// touched (cget/cset): on this firmware "amixer sget Master" after boot drops
// the codec gain and silences the speaker.
const shadowSaved = "/tmp/ytm-shadow-saved"

var shadowMu sync.Mutex

func shadowGet() string {
	out, err := exec.Command("amixer", "-c0", "cget", "name=ShadowMaster").Output()
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(out), "\n") {
		if i := strings.Index(l, ": values="); i >= 0 {
			return strings.TrimSpace(l[i+len(": values="):])
		}
	}
	return ""
}

func shadowSet(v string) {
	exec.Command("amixer", "-q", "-c0", "cset", "name=ShadowMaster", v).Run()
}

func muteAlice()    { mute(true) }
func restoreAlice() { restore(true) }

// mute and restore are the same with the journal line optional: the brief
// mute around every plain phrase would otherwise flood the journal.
func mute(verbose bool) {
	shadowMu.Lock()
	defer shadowMu.Unlock()
	if _, err := os.Stat(shadowSaved); err == nil {
		return // already muted, by us or by ytm-alice
	}
	v := shadowGet()
	if v == "" || os.WriteFile(shadowSaved, []byte(v), 0600) != nil {
		return
	}
	shadowSet("0,0")
	if verbose {
		logf("Alice muted (ShadowMaster %s -> 0)", v)
	}
}

func restore(verbose bool) {
	shadowMu.Lock()
	defer shadowMu.Unlock()
	raw, err := os.ReadFile(shadowSaved)
	if err != nil {
		return
	}
	v := string(bytes.TrimSpace(raw))
	shadowSet(v)
	os.Remove(shadowSaved)
	if verbose {
		logf("Alice unmuted (ShadowMaster %s)", v)
	}
}
