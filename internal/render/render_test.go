package render

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/LAwLi3tCoding/signalrail/internal/adapter"
	"github.com/LAwLi3tCoding/signalrail/internal/config"
	"github.com/LAwLi3tCoding/signalrail/internal/status"
)

func TestGoldenWidthsNeverWrap(t *testing.T) {
	snapshot, cfg := fixture()
	for _, width := range []int{40, 60, 80, 120, 160} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			got, err := Render(snapshot, cfg, Options{Width: width, Format: "plain", Now: time.Now()})
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := os.ReadFile(fmt.Sprintf("../../testdata/golden/width-%d.txt", width))
			if err != nil {
				t.Fatal(err)
			}
			want := strings.TrimSuffix(string(wantBytes), "\n")
			if got.Output != want {
				t.Fatalf("\ngot:  %q\nwant: %q", got.Output, want)
			}
			if strings.Contains(got.Output, "\n") || VisibleWidth(got.Output) > width {
				t.Fatalf("width=%d output=%q", VisibleWidth(got.Output), got.Output)
			}
		})
	}
}

func TestANSIHasSameVisibleTextAndWidth(t *testing.T) {
	snapshot, cfg := fixture()
	plain, _ := Render(snapshot, cfg, Options{Width: 120, Format: "plain"})
	ansi, _ := Render(snapshot, cfg, Options{Width: 120, Format: "ansi", Color: true})
	if StripANSI(ansi.Output) != plain.Output {
		t.Fatalf("ansi=%q plain=%q", ansi.Output, plain.Output)
	}
	if VisibleWidth(ansi.Output) != VisibleWidth(plain.Output) {
		t.Fatal("ANSI changed visible width")
	}
}

func TestCJKPrivacyAndFreshnessMarkers(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Task.Value.Title = "/Users/alice/客户实现渲染器"
	snapshot.Context.Freshness = status.FreshnessCached
	snapshot.Project.Value.Branch = "private/customer-x"
	cfg.Privacy.SensitiveBranches = []string{"private/*"}
	got, err := Render(snapshot, cfg, Options{Width: 40, Format: "plain", HomeDir: "/Users/alice", UserName: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if VisibleWidth(got.Output) > 40 || strings.Contains(got.Output, "alice") || strings.Contains(got.Output, "customer-x") {
		t.Fatalf("output=%q", got.Output)
	}
	if !strings.Contains(got.Output, "↻") {
		t.Fatalf("missing cached marker: %q", got.Output)
	}
}

func TestPrivacyRedactsAlternatePathSeparators(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Task.Value.Title = `C:\Users\alice\secret\build`
	got, err := Render(snapshot, cfg, Options{Width: 120, Format: "plain", HomeDir: "C:/Users/alice", UserName: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Output, `C:\Users`) || strings.Contains(got.Output, "alice") {
		t.Fatalf("output=%q", got.Output)
	}
}

func TestConfiguredOrderAndExclusionAreHonored(t *testing.T) {
	snapshot, cfg := fixture()
	cfg.Segments = []status.SegmentName{status.SegmentContext, status.SegmentModel}
	got, err := Render(snapshot, cfg, Options{Width: 120, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Output, "CTX 38% left  ◆ GPT-5.5 high") {
		t.Fatalf("output=%q", got.Output)
	}
	if strings.Contains(got.Output, "renderer") || strings.Contains(got.Output, "signalrail") {
		t.Fatalf("excluded segment rendered: %q", got.Output)
	}
}

func TestHostileInputCannotInjectLinesOrANSI(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Task.Value.Title = "build\n\x1b[31mPWN\rnext\tstep"
	got, err := Render(snapshot, cfg, Options{Width: 120, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got.Output, "\n\r\x1b") || !strings.Contains(got.Output, "build PWN next step") {
		t.Fatalf("output=%q", got.Output)
	}
}

func TestASCIIAndNoColorContainNoEscapeOrDecorativeUnicode(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Context.Freshness = status.FreshnessCached
	got, err := Render(snapshot, cfg, Options{Width: 160, Format: "ansi", Color: false, ASCII: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Output, "\x1b") || strings.ContainsAny(got.Output, "◆◇▶✓█░↻…") {
		t.Fatalf("output=%q", got.Output)
	}
}

func TestASCIIAndNoColorGoldens(t *testing.T) {
	snapshot, cfg := fixture()
	for name, opts := range map[string]Options{
		"ascii-160":   {Width: 160, Format: "plain", ASCII: true},
		"nocolor-120": {Width: 120, Format: "ansi", Color: false},
	} {
		got, err := Render(snapshot, cfg, opts)
		if err != nil {
			t.Fatal(err)
		}
		wantBytes, err := os.ReadFile("../../testdata/golden/" + name + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.TrimSuffix(string(wantBytes), "\n"); got.Output != want {
			t.Fatalf("%s: got=%q want=%q", name, got.Output, want)
		}
	}
}

func TestEmojiUsesDisplayCellWidth(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Task.Value.Title = "🚀修复渲染"
	got, err := Render(snapshot, cfg, Options{Width: 24, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if VisibleWidth(got.Output) > 24 {
		t.Fatalf("width=%d output=%q", VisibleWidth(got.Output), got.Output)
	}
}

func TestPathologicalNarrowWidthsAndUnavailableSegments(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Model.Value.Name = strings.Repeat("LONG", 20)
	snapshot.Context.Value.LeftLabel = strings.Repeat("9", 20) + "%"
	snapshot.Task.Confidence = status.ConfidenceUnavailable
	cfg.Segments = []status.SegmentName{status.SegmentTask, status.SegmentModel, status.SegmentContext}
	for width := 1; width <= 24; width++ {
		got, err := Render(snapshot, cfg, Options{Width: width, Format: "plain"})
		if err != nil {
			t.Fatalf("width %d: %v", width, err)
		}
		if VisibleWidth(got.Output) > width || strings.Contains(got.Output, "Build renderer") {
			t.Fatalf("width=%d output=%q", width, got.Output)
		}
	}
}

func TestNarrowEvictionPreservesModelThenTask(t *testing.T) {
	snapshot, cfg := fixture()
	cfg.Segments = []status.SegmentName{status.SegmentModel, status.SegmentProject, status.SegmentTask, status.SegmentProgress, status.SegmentContext, status.SegmentCost}
	got, err := Render(snapshot, cfg, Options{Width: 24, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got.Included, status.SegmentModel) || !contains(got.Included, status.SegmentTask) || contains(got.Included, status.SegmentContext) {
		t.Fatalf("included=%v output=%q", got.Included, got.Output)
	}
	got, err = Render(snapshot, cfg, Options{Width: 8, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Included) != 1 || got.Included[0] != status.SegmentModel {
		t.Fatalf("included=%v output=%q", got.Included, got.Output)
	}
}

func TestSensitiveBranchRedactionAndDeterminism(t *testing.T) {
	snapshot, cfg := fixture()
	snapshot.Project.Value.Branch = "private/customer-x"
	cfg.Privacy.SensitiveBranches = []string{"private/*"}
	first, err := Render(snapshot, cfg, Options{Width: 120, Format: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := Render(snapshot, cfg, Options{Width: 120, Format: "plain"})
	if first.Output != second.Output {
		t.Fatalf("nondeterministic: %q != %q", first.Output, second.Output)
	}
	if strings.Contains(first.Output, "customer-x") || !strings.Contains(first.Output, "[redacted]") {
		t.Fatalf("output=%q", first.Output)
	}
}

func TestFreshnessAndConfidenceMarkers(t *testing.T) {
	snapshot, cfg := fixture()
	cfg.Segments = []status.SegmentName{status.SegmentModel, status.SegmentContext, status.SegmentCost}
	snapshot.Model.Freshness = status.FreshnessStale
	snapshot.Context.Freshness = status.FreshnessDegraded
	got, _ := Render(snapshot, cfg, Options{Width: 160, Format: "plain"})
	if strings.Count(got.Output, "!") != 2 || !strings.Contains(got.Output, "$1.24~") {
		t.Fatalf("output=%q", got.Output)
	}
}

func fixture() (status.Snapshot, config.Config) {
	now := time.Now()
	exact := func() status.Source { return status.Source{Provider: "fixture"} }
	s := status.Snapshot{
		Model:   status.Datum[status.Model]{Value: status.Model{Name: "GPT-5.5", Effort: "high"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact(), ObservedAt: now},
		Project: status.Datum[status.Project]{Value: status.Project{Name: "signalrail", Branch: "main"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact(), ObservedAt: now},
		Task:    status.Datum[status.Task]{Value: status.Task{Title: "Build renderer", Step: 3, TotalSteps: 7, State: "active"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact(), ObservedAt: now},
		Context: status.Datum[status.Context]{Value: status.Context{UsedPercent: 62, LeftLabel: "38%"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact(), ObservedAt: now},
		Cost:    status.Datum[status.Cost]{Value: status.Cost{Amount: 1.24, Currency: "USD"}, Confidence: status.ConfidenceEstimated, Freshness: status.FreshnessFresh, Source: exact(), ObservedAt: now},
	}
	cfg := config.Config{Segments: []status.SegmentName{status.SegmentModel, status.SegmentProject, status.SegmentTask, status.SegmentProgress, status.SegmentContext, status.SegmentCost}, Privacy: config.Privacy{RedactUser: true, RedactPaths: true}, Cost: config.Cost{Enabled: true, Currency: "USD"}}
	return s, cfg
}

func BenchmarkRender120(b *testing.B) {
	snapshot, cfg := fixture()
	opts := Options{Width: 120, Format: "plain"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Render(snapshot, cfg, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func TestRenderP95PerformanceBudget(t *testing.T) {
	snapshot, cfg := fixture()
	opts := Options{Width: 120, Format: "plain"}
	durations := make([]time.Duration, 200)
	for i := range durations {
		started := time.Now()
		if _, err := Render(snapshot, cfg, opts); err != nil {
			t.Fatal(err)
		}
		durations[i] = time.Since(started)
	}
	if got, want := percentile95(durations), 20*time.Millisecond; got >= want {
		t.Fatalf("cached render p95 %s exceeds budget %s", got, want)
	}
}

func TestParseAndRenderP95PerformanceBudget(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/claude/full.json")
	if err != nil {
		t.Fatal(err)
	}
	_, cfg := fixture()
	opts := Options{Width: 120, Format: "plain"}
	durations := make([]time.Duration, 100)
	for i := range durations {
		started := time.Now()
		snapshot, err := adapter.ParseClaude(bytes.NewReader(payload), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Render(snapshot, cfg, opts); err != nil {
			t.Fatal(err)
		}
		durations[i] = time.Since(started)
	}
	if got, want := percentile95(durations), 50*time.Millisecond; got >= want {
		t.Fatalf("parse-and-render p95 %s exceeds budget %s", got, want)
	}
}

func percentile95(durations []time.Duration) time.Duration {
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	return durations[(len(durations)*95+99)/100-1]
}
