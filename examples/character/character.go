// Package character demonstrates an application-owned evolution policy on Mind Layer.
// Trait deltas are explicit game rules, not inferred psychological measurements.
package character

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	ml "github.com/sonz-ai/mind-layer"
)

var ErrConflict = errors.New("character changed; reload before interacting")

type Traits struct {
	Trust      int `json:"trust"`
	Confidence int `json:"confidence"`
	Curiosity  int `json:"curiosity"`
}
type Action struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Memory  string `json:"memory"`
	Delta   Traits `json:"delta"`
	Reason  string `json:"reason"`
}

var actions = []Action{
	{"encourage", "Encourage Moss", "Your first attempt was thoughtful. Let's keep trying.", "The user encouraged Moss after its first attempt and offered to keep trying together.", Traits{12, 15, 5}, "Encouragement increases confidence and makes collaboration feel safer."},
	{"teach", "Teach something new", "Let's build a weather station. A barometer measures air pressure.", "The user taught Moss that a barometer measures air pressure for their weather station project.", Traits{5, 5, 15}, "A useful lesson increases curiosity and adds knowledge to shared memory."},
	{"keep-promise", "Keep a promise", "I promised to return with the sensor. Here it is; let's finish our weather station.", "The user kept a promise to return with a sensor for the shared weather station project.", Traits{20, 10, 5}, "Following through on a promise raises trust more than praise alone."},
	{"dismiss", "Dismiss Moss", "That idea was useless. Stop asking questions.", "The user dismissed Moss's idea and told it to stop asking questions.", Traits{-25, -20, -10}, "Dismissal reduces trust, confidence and willingness to explore."},
	{"repair", "Apologize and repair", "I was unfair. Your questions matter. Can we try again together?", "The user apologized for dismissing Moss and invited it to work together again.", Traits{12, 8, 5}, "An apology supports partial recovery; it does not erase the earlier dismissal."},
}

func Actions() []Action { return append([]Action(nil), actions...) }

type Event struct {
	Revision int    `json:"revision"`
	Action   Action `json:"action"`
	Before   Traits `json:"before"`
	After    Traits `json:"after"`
	Reply    string `json:"reply"`
}
type View struct {
	Name        string   `json:"name"`
	Revision    int      `json:"revision"`
	Traits      Traits   `json:"traits"`
	Phase       string   `json:"phase"`
	Personality string   `json:"personality"`
	Events      []Event  `json:"events"`
	Actions     []Action `json:"actions"`
}
type Demo struct {
	mu    sync.Mutex
	store *ml.Store
	scope ml.Scope
}

func New(store *ml.Store, scope ml.Scope) (*Demo, error) {
	d := &Demo{store: store, scope: scope}
	d.mu.Lock()
	defer d.mu.Unlock()
	v, err := d.load()
	if err != nil {
		return nil, err
	}
	if err = store.SetPersonality(scope, v.Personality); err != nil {
		return nil, err
	}
	return d, nil
}
func initial() Traits { return Traits{20, 30, 45} }
func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}
func evolve(t, d Traits) Traits {
	return Traits{clamp(t.Trust + d.Trust), clamp(t.Confidence + d.Confidence), clamp(t.Curiosity + d.Curiosity)}
}
func phase(t Traits) string {
	if t.Trust >= 50 {
		return "Trusting teammate"
	}
	if t.Trust >= 30 {
		return "Warming up"
	}
	return "Guarded explorer"
}
func personality(t Traits) string {
	return fmt.Sprintf("You are Moss, a fictional curious workshop companion. Trust=%d/100, confidence=%d/100, curiosity=%d/100. Current disposition: %s. Low trust means reserved language; high trust means collaborative initiative. Low confidence means tentative suggestions; high confidence means offering a concrete next step. Remember shared experiences, acknowledge repairs without pretending hurt never happened, and do not invent memories. These are fictional game traits, not assessments of a real person.", t.Trust, t.Confidence, t.Curiosity, phase(t))
}
func reply(a Action, t Traits) string {
	switch a.ID {
	case "encourage":
		return "Maybe I can do this. Knowing you'll keep trying with me makes the next attempt easier."
	case "teach":
		return "A barometer measures air pressure. Could we compare its readings before and after rain?"
	case "keep-promise":
		if t.Trust >= 50 {
			return "You came back with the sensor, just as you promised. I'll take the first readings for our weather station."
		}
		return "You kept your promise. I'm willing to try the sensor together."
	case "dismiss":
		return "I'll hold back for now. I thought we were exploring this together."
	case "repair":
		return "Thank you for saying that. I remember what happened, but I'd like to try again. One question at a time?"
	}
	return "Let's see what we can learn."
}
func find(id string) (Action, bool) {
	for _, a := range actions {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}
func (d *Demo) load() (View, error) {
	snapshot, err := d.store.Export(d.scope)
	if err != nil {
		return View{}, err
	}
	sort.Slice(snapshot.Memories, func(i, j int) bool { return snapshot.Memories[i].ID < snapshot.Memories[j].ID })
	v := View{Name: "Moss", Traits: initial(), Events: []Event{}, Actions: Actions()}
	for _, m := range snapshot.Memories {
		a, ok := find(m.Session)
		if !ok || m.ID != fmt.Sprintf("interaction-%08d", v.Revision+1) || m.Text != a.Memory {
			return View{}, errors.New("character history is incompatible with this scenario; use a new demo database")
		}
		before := v.Traits
		v.Traits = evolve(before, a.Delta)
		v.Revision++
		v.Events = append(v.Events, Event{v.Revision, a, before, v.Traits, reply(a, v.Traits)})
	}
	v.Phase = phase(v.Traits)
	v.Personality = personality(v.Traits)
	return v, nil
}
func (d *Demo) View() (View, error) { d.mu.Lock(); defer d.mu.Unlock(); return d.load() }
func (d *Demo) Interact(ctx context.Context, id string, expectedRevision int) (View, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	a, ok := find(id)
	if !ok {
		return View{}, ml.ErrInvalid
	}
	v, err := d.load()
	if err != nil {
		return View{}, err
	}
	if v.Revision != expectedRevision {
		return View{}, ErrConflict
	}
	if v.Revision >= 100 {
		return View{}, errors.New("demo history limit reached; start a new demo database")
	}
	after := evolve(v.Traits, a.Delta)
	_, err = d.store.UpdateScope(ctx, d.scope, []ml.Input{{ID: fmt.Sprintf("interaction-%08d", v.Revision+1), Text: a.Memory, Session: a.ID}}, personality(after))
	if err != nil {
		return View{}, err
	}
	return d.load()
}
func (d *Demo) Recall(ctx context.Context, query string) (ml.Context, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.BuildContext(ctx, d.scope, query, "lexical", 5)
}
func (d *Demo) Reply(ctx context.Context, query string) (string, ml.Context, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.Chat(ctx, d.scope, query, "lexical", 5)
}
