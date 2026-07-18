package ops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo makes an empty git repo in a temp dir and chdirs into it for the
// duration of the test. Every ops function works on "the current repo", so
// tests have to move rather than pass a path.
func newRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	// macOS temp dirs live under /var, a symlink to /private/var. git reports
	// the resolved path, so resolve up front or path comparisons drift.
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func remove(t *testing.T, name string) {
	t.Helper()
	if err := os.Remove(name); err != nil {
		t.Fatalf("remove %s: %v", name, err)
	}
}

// gitOut runs git and returns its trimmed stdout, failing the test on error.
func gitOut(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

func TestChangesCategorises(t *testing.T) {
	newRepo(t)
	write(t, "kept.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	write(t, "new.txt", "brand new")
	write(t, "kept.txt", "edited")
	write(t, "gone.txt", "doomed")
	if _, err := SaveAll("second"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	remove(t, "gone.txt")
	write(t, "kept.txt", "edited again")
	write(t, "fresh.txt", "untracked")

	changes, err := Changes()
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}

	got := map[string]ChangeKind{}
	for _, c := range changes {
		got[c.Path] = c.Kind
	}
	want := map[string]ChangeKind{
		"gone.txt":  ChangeRemoved,
		"kept.txt":  ChangeModified,
		"fresh.txt": ChangeNew,
	}
	for path, kind := range want {
		if got[path] != kind {
			t.Errorf("%s: got kind %v (%s), want %v (%s)",
				path, got[path], got[path].Label(), kind, kind.Label())
		}
	}
	if len(changes) != len(want) {
		t.Errorf("got %d changes, want %d: %+v", len(changes), len(want), changes)
	}
}

// TestSaveWithDeletedFile covers the case that used to wedge gitle entirely:
// once a deletion was staged, `commit -- <path>` failed with "pathspec did not
// match any files" and no further save could ever succeed.
func TestSaveWithDeletedFile(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	write(t, "b.txt", "two")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	remove(t, "b.txt")
	write(t, "a.txt", "one edited")

	changes, err := Changes()
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}
	paths := Paths(changes)

	// Stage, then abandon the save — this is what leaves a staged deletion
	// behind, exactly as an interrupted `gitle save` does.
	if err := Stage(paths); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	// Now save for real. This must succeed despite the staged deletion.
	result, err := Save("with a deletion", paths)
	if err != nil {
		t.Fatalf("Save after staged deletion: %v", err)
	}
	if result.Leftover {
		t.Errorf("expected a clean tree after saving everything, got leftover changes")
	}

	if files := gitOut(t, "show", "--name-only", "--pretty=format:", "HEAD"); !strings.Contains(files, "b.txt") {
		t.Errorf("commit should record b.txt's deletion, got files: %q", files)
	}
	if _, err := os.Stat("b.txt"); !os.IsNotExist(err) {
		t.Errorf("b.txt should still be gone from the working tree")
	}
}

// TestStageIsExact checks that unticking a file in the checklist really does
// keep it out of the save.
func TestStageIsExact(t *testing.T) {
	newRepo(t)
	write(t, "keep.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	write(t, "keep.txt", "edited")
	write(t, "skip.txt", "not this one")

	if _, err := Save("only the picked file", []string{"keep.txt"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	files := gitOut(t, "show", "--name-only", "--pretty=format:", "HEAD")
	if strings.Contains(files, "skip.txt") {
		t.Errorf("unpicked file was committed anyway, got: %q", files)
	}
	if !strings.Contains(files, "keep.txt") {
		t.Errorf("picked file missing from commit, got: %q", files)
	}
}

func TestSaveRejectsEmptyMessage(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := Save("", []string{"a.txt"}); err != ErrNoMessage {
		t.Errorf("got %v, want ErrNoMessage", err)
	}
}

func TestUndoLastSaveKeepsChanges(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	write(t, "a.txt", "two")
	if _, err := SaveAll("second"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	if err := UndoLastSave(); err != nil {
		t.Fatalf("UndoLastSave: %v", err)
	}

	if subject := gitOut(t, "log", "-1", "--pretty=%s"); subject != "first" {
		t.Errorf("HEAD is %q, want the save before last (%q)", subject, "first")
	}
	content, err := os.ReadFile("a.txt")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "two" {
		t.Errorf("file content is %q, want %q — undo must keep the changes", content, "two")
	}
}

// TestUndoFirstSave covers the unborn-HEAD path: undoing the very first save
// has no parent commit to reset to.
func TestUndoFirstSave(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	if err := UndoLastSave(); err != nil {
		t.Fatalf("UndoLastSave: %v", err)
	}

	s, err := CurrentStatus()
	if err != nil {
		t.Fatalf("CurrentStatus: %v", err)
	}
	if s.HasCommits {
		t.Errorf("repo should have no saves left after undoing the only one")
	}
	if _, err := os.Stat("a.txt"); err != nil {
		t.Errorf("a.txt should survive the undo: %v", err)
	}
}

func TestUndoNothingSaved(t *testing.T) {
	newRepo(t)
	if err := UndoLastSave(); err != ErrNothingToUndo {
		t.Errorf("got %v, want ErrNothingToUndo", err)
	}
	if _, err := LastSaveMessage(); err != ErrNothingToUndo {
		t.Errorf("got %v, want ErrNothingToUndo", err)
	}
}

func TestDiscard(t *testing.T) {
	newRepo(t)
	write(t, "tracked.txt", "original")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	write(t, "tracked.txt", "meddled with")
	write(t, "untracked.txt", "junk")

	if err := Discard(); err != nil {
		t.Fatalf("Discard: %v", err)
	}

	content, err := os.ReadFile("tracked.txt")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "original" {
		t.Errorf("tracked file is %q, want it reverted to %q", content, "original")
	}
	if _, err := os.Stat("untracked.txt"); !os.IsNotExist(err) {
		t.Errorf("untracked file should have been cleaned away")
	}
}

func TestCurrentStatus(t *testing.T) {
	dir := newRepo(t)

	s, err := CurrentStatus()
	if err != nil {
		t.Fatalf("CurrentStatus: %v", err)
	}
	if s.HasCommits {
		t.Errorf("fresh repo should have no saves")
	}
	if s.Name != filepath.Base(dir) {
		t.Errorf("repo name is %q, want %q", s.Name, filepath.Base(dir))
	}
	if s.HasRemote || s.HasUpstream {
		t.Errorf("fresh repo should not be online")
	}
	if s.Conflicted() {
		t.Errorf("fresh repo should not be mid-conflict")
	}

	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if s, err = CurrentStatus(); err != nil {
		t.Fatalf("CurrentStatus: %v", err)
	}
	if !s.HasCommits {
		t.Errorf("repo should have a save now")
	}
	if s.Branch != "main" {
		t.Errorf("branch is %q, want main", s.Branch)
	}
	if len(s.Changes) != 0 {
		t.Errorf("tree should be clean, got %+v", s.Changes)
	}
}

func TestBranches(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	if err := NewBranch("feature"); err != nil {
		t.Fatalf("NewBranch: %v", err)
	}
	if err := NewBranch("feature"); err != ErrBranchExists {
		t.Errorf("got %v, want ErrBranchExists", err)
	}
	if err := Switch("nope"); err != ErrBranchMissing {
		t.Errorf("got %v, want ErrBranchMissing", err)
	}

	branches, err := Branches()
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	current := ""
	names := map[string]bool{}
	for _, b := range branches {
		names[b.Name] = true
		if b.Current {
			current = b.Name
		}
	}
	if current != "feature" {
		t.Errorf("current branch is %q, want feature", current)
	}
	if !names["main"] || !names["feature"] {
		t.Errorf("expected both branches, got %+v", branches)
	}

	if err := Switch("main"); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if got := gitOut(t, "rev-parse", "--abbrev-ref", "HEAD"); got != "main" {
		t.Errorf("on branch %q, want main", got)
	}
}

func TestHistory(t *testing.T) {
	newRepo(t)
	if commits, err := History(0); err != nil || len(commits) != 0 {
		t.Errorf("empty repo: got %d commits, %v; want none", len(commits), err)
	}

	write(t, "a.txt", "one")
	if _, err := SaveAll("first save"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	write(t, "a.txt", "two")
	// A subject containing characters that could confuse field splitting.
	if _, err := SaveAll("second: with, punctuation | and pipes"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	commits, err := History(0)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2: %+v", len(commits), commits)
	}
	if commits[0].Subject != "second: with, punctuation | and pipes" {
		t.Errorf("newest subject is %q", commits[0].Subject)
	}
	if commits[1].Subject != "first save" {
		t.Errorf("oldest subject is %q", commits[1].Subject)
	}
	if commits[0].Author != "Test" {
		t.Errorf("author is %q, want Test", commits[0].Author)
	}
	if commits[0].Hash == "" || commits[0].When == "" {
		t.Errorf("hash and relative date should be populated: %+v", commits[0])
	}

	if limited, err := History(1); err != nil || len(limited) != 1 {
		t.Errorf("History(1): got %d commits, %v; want 1", len(limited), err)
	}
}

func TestFileDiffIncludesUntracked(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	write(t, "brand-new.txt", "never seen before")
	diff, err := FileDiff("brand-new.txt")
	if err != nil {
		t.Fatalf("FileDiff: %v", err)
	}
	// A file git has never seen produces nothing from a plain `git diff`, so
	// this is really checking the --no-index fallback fires.
	if !strings.Contains(diff, "never seen before") {
		t.Errorf("diff of an untracked file should show its contents, got:\n%s", diff)
	}

	write(t, "a.txt", "one changed")
	if diff, err = FileDiff("a.txt"); err != nil {
		t.Fatalf("FileDiff: %v", err)
	}
	if !strings.Contains(diff, "one changed") {
		t.Errorf("diff of a modified file should show the change, got:\n%s", diff)
	}
}

func TestGrabRefusesDirtyTree(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	write(t, "a.txt", "unsaved edit")

	if err := Grab(); err != ErrUnsavedChanges {
		t.Errorf("got %v, want ErrUnsavedChanges", err)
	}
}

func TestGrabWithoutRemote(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if err := Grab(); err != ErrNoRemote {
		t.Errorf("got %v, want ErrNoRemote", err)
	}
}

func TestSendWithoutCommits(t *testing.T) {
	newRepo(t)
	if _, err := Send(SendOptions{}); err != ErrNothingToSend {
		t.Errorf("got %v, want ErrNothingToSend", err)
	}
}

func TestSendWithoutRemote(t *testing.T) {
	newRepo(t)
	write(t, "a.txt", "one")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if _, err := Send(SendOptions{}); err != ErrNoRemote {
		t.Errorf("got %v, want ErrNoRemote", err)
	}
}

func TestClassifyPush(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   SendProblem
	}{
		{"rejected", "! [rejected]        main -> main (fetch first)", SendRejected},
		{"non fast forward", "Updates were rejected because of a non-fast-forward", SendRejected},
		{"auth", "fatal: Authentication failed for 'https://github.com/x/y'", SendAuth},
		{"no terminal prompt", "fatal: could not read Username for 'https://github.com'", SendAuth},
		{"permission", "ERROR: Permission denied (publickey)", SendAuth},
		{"other", "fatal: the remote end hung up unexpectedly\nsecond line", SendUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyPush(tc.stderr)
			if got.Problem != tc.want {
				t.Errorf("got problem %v, want %v", got.Problem, tc.want)
			}
			if tc.want == SendUnknown && strings.Contains(got.Detail, "\n") {
				t.Errorf("detail should be a single line, got %q", got.Detail)
			}
		})
	}
}

func TestScanRisks(t *testing.T) {
	newRepo(t)
	write(t, ".env", "SECRET=hunter2")
	write(t, "server.pem", "-----BEGIN PRIVATE KEY-----")
	write(t, "normal.txt", "nothing to see")

	risks := ScanRisks([]string{".env", "server.pem", "normal.txt"})
	if !risks.Any() {
		t.Fatalf("expected risks to be flagged")
	}
	if len(risks.Secrets) != 2 {
		t.Errorf("got %d secrets, want 2: %v", len(risks.Secrets), risks.Secrets)
	}
	if len(risks.Large) != 0 {
		t.Errorf("nothing here is large, got %v", risks.Large)
	}

	if clean := ScanRisks([]string{"normal.txt"}); clean.Any() {
		t.Errorf("ordinary file should not be flagged: %+v", clean)
	}
}

func TestScanRisksLargeFile(t *testing.T) {
	newRepo(t)
	big := make([]byte, largeFileBytes+1)
	if err := os.WriteFile("big.bin", big, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	risks := ScanRisks([]string{"big.bin"})
	if len(risks.Large) != 1 {
		t.Fatalf("got %d large files, want 1", len(risks.Large))
	}
	if risks.Large[0].Size != "10.0 MB" {
		t.Errorf("size rendered as %q, want %q", risks.Large[0].Size, "10.0 MB")
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		512:            "512 B",
		2048:           "2.0 KB",
		5 * 1048576:    "5.0 MB",
		3 * 1073741824: "3.0 GB",
	}
	for n, want := range cases {
		if got := HumanSize(n); got != want {
			t.Errorf("HumanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"only line":            "only line",
		"first\nsecond\nthird": "first",
		"  padded  \nnext":     "padded",
		"":                     "",
	}
	for in, want := range cases {
		if got := FirstLine(in); got != want {
			t.Errorf("FirstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// makeConflict builds a repo stopped mid-merge with a real clash in a.txt.
func makeConflict(t *testing.T) {
	t.Helper()
	newRepo(t)
	write(t, "a.txt", "line one\nshared\nline three\n")
	if _, err := SaveAll("first"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	if err := NewBranch("other"); err != nil {
		t.Fatalf("NewBranch: %v", err)
	}
	write(t, "a.txt", "line one\nTHEIRS\nline three\n")
	if _, err := SaveAll("their change"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	if err := Switch("main"); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	write(t, "a.txt", "line one\nOURS\nline three\n")
	if _, err := SaveAll("our change"); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	// Expected to fail — that's the point.
	_ = exec.Command("git", "merge", "other").Run()
}

func TestConflictsDetected(t *testing.T) {
	makeConflict(t)

	state, err := Conflicts()
	if err != nil {
		t.Fatalf("Conflicts: %v", err)
	}
	if state.Op == 0 {
		t.Fatal("expected an operation in progress")
	}
	if len(state.Files) != 1 || state.Files[0] != "a.txt" {
		t.Fatalf("expected a.txt to be conflicted, got %v", state.Files)
	}
	if state.HeadLabel == "" || state.OtherLabel == "" {
		t.Errorf("both sides need plain-English names, got %q and %q", state.HeadLabel, state.OtherLabel)
	}
}

func TestResolveKeepingEachSide(t *testing.T) {
	cases := []struct {
		name string
		side Side
		want string
	}{
		{"ours", SideOurs, "line one\nOURS\nline three\n"},
		{"theirs", SideTheirs, "line one\nTHEIRS\nline three\n"},
		{"both", SideBoth, "line one\nOURS\nTHEIRS\nline three\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			makeConflict(t)

			f, err := LoadConflictFile("a.txt")
			if err != nil {
				t.Fatalf("LoadConflictFile: %v", err)
			}
			if len(f.Hunks) != 1 {
				t.Fatalf("expected 1 clashing section, got %d", len(f.Hunks))
			}
			if got := strings.Join(f.Hunks[0].Ours, "\n"); got != "OURS" {
				t.Errorf("our side is %q, want OURS", got)
			}
			if got := strings.Join(f.Hunks[0].Theirs, "\n"); got != "THEIRS" {
				t.Errorf("their side is %q, want THEIRS", got)
			}

			if err := f.Resolve([]Side{tc.side}); err != nil {
				t.Fatalf("Resolve: %v", err)
			}

			content, err := os.ReadFile("a.txt")
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if string(content) != tc.want {
				t.Errorf("file is:\n%q\nwant:\n%q", content, tc.want)
			}
			if StillConflicted("a.txt") {
				t.Errorf("markers should be gone")
			}

			// The file must be staged, and finishing must complete the merge.
			left, err := Conflicts()
			if err != nil {
				t.Fatalf("Conflicts: %v", err)
			}
			if len(left.Files) != 0 {
				t.Errorf("nothing should be left unresolved, got %v", left.Files)
			}
			if err := FinishOp(left.Op); err != nil {
				t.Fatalf("FinishOp: %v", err)
			}
			after, err := Conflicts()
			if err != nil {
				t.Fatalf("Conflicts: %v", err)
			}
			if after.Op != 0 {
				t.Errorf("merge should be finished, still in op %v", after.Op)
			}
		})
	}
}

func TestAbortRestoresPreviousState(t *testing.T) {
	makeConflict(t)
	before := gitOut(t, "rev-parse", "HEAD")

	state, err := Conflicts()
	if err != nil {
		t.Fatalf("Conflicts: %v", err)
	}
	if err := AbortOp(state.Op); err != nil {
		t.Fatalf("AbortOp: %v", err)
	}

	after, err := Conflicts()
	if err != nil {
		t.Fatalf("Conflicts: %v", err)
	}
	if after.Op != 0 {
		t.Errorf("operation should be gone after aborting")
	}
	if got := gitOut(t, "rev-parse", "HEAD"); got != before {
		t.Errorf("HEAD moved during an abort: %s -> %s", before, got)
	}
	content, err := os.ReadFile("a.txt")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(content), "OURS") || strings.Contains(string(content), "<<<<<<<") {
		t.Errorf("file should be back to our version, got %q", content)
	}
}

func TestKeepWholeFile(t *testing.T) {
	makeConflict(t)

	if err := KeepWholeFile("a.txt", SideTheirs); err != nil {
		t.Fatalf("KeepWholeFile: %v", err)
	}
	content, err := os.ReadFile("a.txt")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(content), "THEIRS") {
		t.Errorf("expected their version, got %q", content)
	}
	if StillConflicted("a.txt") {
		t.Errorf("markers should be gone")
	}
}

func TestParseConflictsHandlesDiff3Base(t *testing.T) {
	// diff3-style output adds a base section, which gitle discards.
	content := "before\n<<<<<<< HEAD\nours\n||||||| base\noriginal\n=======\ntheirs\n>>>>>>> other\nafter\n"

	hunks, segs, err := parseConflicts(content)
	if err != nil {
		t.Fatalf("parseConflicts: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	if got := strings.Join(hunks[0].Ours, "\n"); got != "ours" {
		t.Errorf("ours = %q, want %q — the base section must not leak in", got, "ours")
	}
	if got := strings.Join(hunks[0].Theirs, "\n"); got != "theirs" {
		t.Errorf("theirs = %q, want %q", got, "theirs")
	}

	rebuilt := rebuild(segs, []string{"chosen"})
	if rebuilt != "before\nchosen\nafter\n" {
		t.Errorf("rebuilt file is %q", rebuilt)
	}
}

func TestParseConflictsRejectsTruncatedMarkers(t *testing.T) {
	for name, content := range map[string]string{
		"no separator": "<<<<<<< HEAD\nours\n",
		"no end":       "<<<<<<< HEAD\nours\n=======\ntheirs\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parseConflicts(content); err != ErrUnparsable {
				t.Errorf("got %v, want ErrUnparsable — a half-written file must not be rewritten", err)
			}
		})
	}
}
