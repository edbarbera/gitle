package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/edbarbera/gitle/internal/gitcmd"
	"github.com/edbarbera/gitle/internal/ui"
	"github.com/spf13/cobra"
)

var fixAdvanced bool

var fixConflictsCmd = &cobra.Command{
	Use:   "fix-conflicts",
	Short: "Walk through conflicts step by step",
	Long: `Walks you through any conflicts left by gitle grab (or a merge), one file at
a time, and lets you pick what to keep — no need to touch the raw markers
yourself.

Use --advanced to go section by section within a file instead of choosing the
whole file at once.`,
	Args:    cobra.NoArgs,
	PreRunE: requireRepo,
	RunE: func(cmd *cobra.Command, args []string) error {
		op := gitcmd.CurrentOp()
		if op == gitcmd.OpNone {
			ui.Info("No conflicts right now — nothing to fix.")
			ui.Hint("If %s ever says something clashed, run this again.", ui.Bold("gitle grab"))
			return nil
		}

		files, err := gitcmd.ConflictedFiles()
		if err != nil {
			return err
		}

		if len(files) > 0 {
			if !ui.IsInteractive() {
				ui.Error("This needs a terminal so it can ask you what to keep.")
				ui.Hint("Run %s in a normal terminal window.", ui.Bold("gitle fix-conflicts"))
				return errSilent
			}

			headLabel, otherLabel := sideLabels(op)
			ui.Warn("%d file(s) need your help:", len(files))
			for _, f := range files {
				ui.Hint("  • %s", f)
			}
			ui.Info("For each one, choose to keep %s, keep %s, or edit it yourself.",
				ui.Bold(headLabel), ui.Bold(otherLabel))

			allResolved := true
			for _, f := range files {
				resolved, aborted := resolveFile(f, headLabel, otherLabel, fixAdvanced)
				if aborted {
					return abortOp(op)
				}
				if !resolved {
					allResolved = false
				}
			}
			if !allResolved {
				ui.Info("Some files still need attention — run %s again when ready.", ui.Bold("gitle fix-conflicts"))
				return nil
			}
		}

		return finishOp(op)
	},
}

// sideLabels names the two sides of a conflict in plain English. Which side
// is "yours" depends on the operation: in a merge/cherry-pick, HEAD is your
// current work; in a rebase your commits are the ones being replayed on top
// of HEAD, so the meaning flips.
func sideLabels(op gitcmd.OpKind) (head, other string) {
	switch op {
	case gitcmd.OpRebase:
		return "what's already there", "your changes"
	case gitcmd.OpCherryPick:
		return "your version", "the commit you're bringing in"
	default: // merge
		return "your version", "the version you grabbed"
	}
}

// resolveFile walks the user through one conflicted file and stages it once
// resolved. It returns whether the file ended up resolved, and whether the
// user asked to abort the whole operation.
func resolveFile(path, headLabel, otherLabel string, advanced bool) (resolved, aborted bool) {
	fmt.Println()
	ui.Plain("%s", ui.Bold(path))

	if advanced {
		return resolveFileAdvanced(path, headLabel, otherLabel)
	}

	options := []string{
		"Keep " + headLabel,
		"Keep " + otherLabel,
		"Edit it myself",
		"Skip for now",
		"Stop and undo everything",
	}
	switch ui.Choose("What do you want to do with this file?", options) {
	case 0:
		if err := gitcmd.CheckoutOurs(path); err != nil {
			ui.Error("%s", err)
			return false, false
		}
	case 1:
		if err := gitcmd.CheckoutTheirs(path); err != nil {
			ui.Error("%s", err)
			return false, false
		}
	case 2:
		ui.Info("Opening %s — remove the <<<<<<<, ======= and >>>>>>> lines, keep what you want, then save and close.", path)
		if err := ui.OpenEditor(path); err != nil {
			ui.Error("Couldn't open your editor.")
			return false, false
		}
		if stillConflicted(path) {
			ui.Warn("Still see conflict markers in %s — leaving it for now.", path)
			return false, false
		}
	case 4:
		return false, true
	default: // 3 (skip) or -1 (no answer / cancelled)
		return false, false
	}

	if err := gitcmd.StageFile(path); err != nil {
		ui.Error("%s", err)
		return false, false
	}
	ui.Success("%s resolved.", path)
	return true, false
}

// stillConflicted reports whether path still has unresolved conflict markers.
func stillConflicted(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "<<<<<<< ")
}

// conflictBlock is one <<<<<<< ... ======= ... >>>>>>> section of a file.
type conflictBlock struct{ ours, theirs []string }

// segment is either a literal chunk of the file or a placeholder for one
// conflictBlock, kept in original order so the file can be rebuilt.
type segment struct {
	text       string
	isConflict bool
	blockIdx   int
}

// resolveFileAdvanced walks the user through a file's conflicts one section
// at a time, letting them keep one side, both sides, or hand-edit just that
// section.
func resolveFileAdvanced(path, headLabel, otherLabel string) (resolved, aborted bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		ui.Error("%s", err)
		return false, false
	}
	blocks, segs, err := parseConflicts(string(data))
	if err != nil {
		ui.Error("Couldn't make sense of the conflicts in %s.", path)
		ui.Hint("Try again without --advanced, or choose \"Edit it myself\".")
		return false, false
	}
	if len(blocks) == 0 {
		if err := gitcmd.StageFile(path); err != nil {
			ui.Error("%s", err)
			return false, false
		}
		return true, false
	}

	ui.Info("%d section(s) to decide on.", len(blocks))
	resolvedBlocks := make([]string, len(blocks))
	for i, b := range blocks {
		fmt.Println()
		ui.Plain("%s", ui.Dim(fmt.Sprintf("Section %d of %d", i+1, len(blocks))))
		ui.Plain("%s:", ui.Bold(headLabel))
		printLines(b.ours)
		ui.Plain("%s:", ui.Bold(otherLabel))
		printLines(b.theirs)

		options := []string{
			"Keep " + headLabel,
			"Keep " + otherLabel,
			"Keep both",
			"Edit this bit myself",
			"Stop and undo everything",
		}
		switch ui.Choose("Which do you want here?", options) {
		case 0:
			resolvedBlocks[i] = strings.Join(b.ours, "\n")
		case 1:
			resolvedBlocks[i] = strings.Join(b.theirs, "\n")
		case 2:
			resolvedBlocks[i] = strings.Join(append(append([]string{}, b.ours...), b.theirs...), "\n")
		case 3:
			edited, err := editSnippet(b.ours, b.theirs)
			if err != nil {
				ui.Error("Couldn't open your editor.")
				return false, false
			}
			resolvedBlocks[i] = edited
		case 4:
			return false, true
		default: // -1: no answer / cancelled
			return false, false
		}
	}

	final := rebuild(segs, resolvedBlocks)
	if err := os.WriteFile(path, []byte(final), 0o644); err != nil {
		ui.Error("%s", err)
		return false, false
	}
	if err := gitcmd.StageFile(path); err != nil {
		ui.Error("%s", err)
		return false, false
	}
	ui.Success("%s resolved.", path)
	return true, false
}

// printLines prints file lines indented, for showing a conflict's contents.
func printLines(lines []string) {
	if len(lines) == 0 {
		ui.Plain("  %s", ui.Dim("(nothing)"))
		return
	}
	for _, l := range lines {
		ui.Plain("  %s", l)
	}
}

// parseConflicts splits file content into literal segments and conflict
// blocks, in order, so the file can be reconstructed after each block is
// resolved. It understands the optional diff3 "|||||||" base section but
// discards it — gitle only ever shows the two sides, never the base.
func parseConflicts(content string) ([]conflictBlock, []segment, error) {
	lines := strings.Split(content, "\n")
	var blocks []conflictBlock
	var segs []segment
	var cur []string

	i := 0
	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "<<<<<<< ") {
			cur = append(cur, lines[i])
			i++
			continue
		}

		if len(cur) > 0 {
			segs = append(segs, segment{text: strings.Join(cur, "\n")})
			cur = nil
		}
		i++

		var ours []string
		for i < len(lines) && !strings.HasPrefix(lines[i], "=======") && !strings.HasPrefix(lines[i], "||||||| ") {
			ours = append(ours, lines[i])
			i++
		}
		if i < len(lines) && strings.HasPrefix(lines[i], "||||||| ") {
			for i < len(lines) && !strings.HasPrefix(lines[i], "=======") {
				i++
			}
		}
		if i >= len(lines) {
			return nil, nil, errors.New("missing ======= marker")
		}
		i++ // skip =======

		var theirs []string
		for i < len(lines) && !strings.HasPrefix(lines[i], ">>>>>>> ") {
			theirs = append(theirs, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, nil, errors.New("missing >>>>>>> marker")
		}
		i++ // skip >>>>>>> line

		blocks = append(blocks, conflictBlock{ours: ours, theirs: theirs})
		segs = append(segs, segment{isConflict: true, blockIdx: len(blocks) - 1})
	}
	if len(cur) > 0 {
		segs = append(segs, segment{text: strings.Join(cur, "\n")})
	}
	return blocks, segs, nil
}

// rebuild reassembles a file's content from its segments, substituting each
// conflict placeholder with its resolved text.
func rebuild(segs []segment, resolvedBlocks []string) string {
	var b strings.Builder
	for idx, s := range segs {
		if s.isConflict {
			b.WriteString(resolvedBlocks[s.blockIdx])
		} else {
			b.WriteString(s.text)
		}
		if idx < len(segs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// editSnippet lets the user hand-write the resolution for one conflict
// section in their editor, showing both sides as commented-out reference.
func editSnippet(ours, theirs []string) (string, error) {
	f, err := os.CreateTemp("", "gitle-conflict-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())

	var content strings.Builder
	content.WriteString("# Write what this section should say below.\n")
	content.WriteString("# Lines starting with # are ignored.\n")
	content.WriteString("#\n# --- your version ---\n")
	for _, l := range ours {
		content.WriteString("# " + l + "\n")
	}
	content.WriteString("#\n# --- their version ---\n")
	for _, l := range theirs {
		content.WriteString("# " + l + "\n")
	}
	content.WriteString("\n")
	if _, err := f.WriteString(content.String()); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	if err := ui.OpenEditor(f.Name()); err != nil {
		return "", err
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	return strings.Trim(strings.Join(out, "\n"), "\n"), nil
}

// abortOp cancels the in-progress operation and restores the working tree to
// how it was before it started.
func abortOp(op gitcmd.OpKind) error {
	ui.Warn("Undoing and going back to how things were...")
	var err error
	switch op {
	case gitcmd.OpMerge:
		err = gitcmd.AbortMerge()
	case gitcmd.OpRebase:
		err = gitcmd.AbortRebase()
	case gitcmd.OpCherryPick:
		err = gitcmd.AbortCherryPick()
	}
	if err != nil {
		ui.Error("Couldn't undo automatically.")
		return errSilent
	}
	ui.Success("Back to where you started — nothing was lost.")
	return nil
}

// finishOp completes the in-progress operation once every file is resolved.
func finishOp(op gitcmd.OpKind) error {
	ui.Info("Finishing up...")
	var stderr string
	var err error
	switch op {
	case gitcmd.OpMerge:
		stderr, err = gitcmd.ContinueMerge()
	case gitcmd.OpRebase:
		stderr, err = gitcmd.ContinueRebase()
	case gitcmd.OpCherryPick:
		stderr, err = gitcmd.ContinueCherryPick()
	}
	if err != nil {
		ui.Error("Couldn't finish automatically.")
		if msg := firstLine(stderr); msg != "" {
			ui.Hint("git said: %s", msg)
		}
		ui.Hint("Run %s again once that's sorted.", ui.Bold("gitle fix-conflicts"))
		return errSilent
	}

	ui.Success("All conflicts resolved!")
	if gitcmd.HasUpstream() {
		ui.Hint("Send it online with %s.", ui.Bold("gitle send"))
	}
	return nil
}

func init() {
	fixConflictsCmd.Flags().BoolVar(&fixAdvanced, "advanced", false, "resolve conflicts section by section instead of whole files")
	rootCmd.AddCommand(fixConflictsCmd)
}
