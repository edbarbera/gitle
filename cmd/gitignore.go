package cmd

import "os"

// gitignoreCommon is added for every project: OS cruft, logs, and — most
// importantly — common secret files that should never be shared.
const gitignoreCommon = `# OS files
.DS_Store
Thumbs.db

# Logs
*.log

# Secrets — never share these
.env
.env.local
.env.*.local
`

// projectType is a detected kind of project used to pick a .gitignore template.
type projectType struct {
	name    string // human label, e.g. "Node.js"
	ignore  string // template body (before the common section)
	matched bool
}

// detectProject sniffs the current folder for tell-tale files and returns a
// fitting .gitignore starting point. Falls back to a generic one.
func detectProject() projectType {
	switch {
	case exists("package.json"):
		return projectType{"Node.js", "# Node\nnode_modules/\ndist/\nbuild/\ncoverage/\n", true}
	case exists("requirements.txt") || exists("pyproject.toml") || exists("setup.py"):
		return projectType{"Python", "# Python\n__pycache__/\n*.pyc\n.venv/\nvenv/\n*.egg-info/\n", true}
	case exists("go.mod"):
		return projectType{"Go", "# Go\n/bin/\n*.exe\n", true}
	case exists("Cargo.toml"):
		return projectType{"Rust", "# Rust\n/target/\n", true}
	case exists("Gemfile"):
		return projectType{"Ruby", "# Ruby\n*.gem\n/.bundle/\nvendor/bundle/\n", true}
	default:
		return projectType{"this folder", "", false}
	}
}

// gitignoreBody builds the full .gitignore contents for a project type.
func (p projectType) gitignoreBody() string {
	if p.ignore == "" {
		return gitignoreCommon
	}
	return p.ignore + "\n" + gitignoreCommon
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
