package cmd

import (
	"os"
	"path/filepath"
)

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
		return projectType{"Node.js", "# Node\nnode_modules/\ndist/\nbuild/\ncoverage/\n.next/\n.turbo/\n", true}
	case exists("requirements.txt") || exists("pyproject.toml") || exists("setup.py") || exists("Pipfile"):
		return projectType{"Python", "# Python\n__pycache__/\n*.pyc\n.venv/\nvenv/\n*.egg-info/\n.pytest_cache/\n.mypy_cache/\n", true}
	case exists("go.mod"):
		return projectType{"Go", "# Go\n/bin/\n*.exe\n*.test\n", true}
	case exists("Cargo.toml"):
		return projectType{"Rust", "# Rust\n/target/\n", true}
	case exists("Gemfile"):
		return projectType{"Ruby", "# Ruby\n*.gem\n/.bundle/\nvendor/bundle/\n", true}
	case exists("pom.xml") || exists("build.gradle") || exists("build.gradle.kts"):
		return projectType{"Java", "# Java\n*.class\ntarget/\nbuild/\n.gradle/\n", true}
	case existsGlob("*.csproj") || existsGlob("*.sln") || existsGlob("*.fsproj"):
		return projectType{".NET", "# .NET\nbin/\nobj/\n*.user\n", true}
	case exists("composer.json"):
		return projectType{"PHP", "# PHP\n/vendor/\ncomposer.phar\n", true}
	case exists("Package.swift") || existsGlob("*.xcodeproj"):
		return projectType{"Swift", "# Swift\n.build/\nDerivedData/\nxcuserdata/\nPackages/\n", true}
	case exists("mix.exs"):
		return projectType{"Elixir", "# Elixir\n_build/\ndeps/\n*.ez\n", true}
	case exists("pubspec.yaml"):
		return projectType{"Dart/Flutter", "# Dart/Flutter\n.dart_tool/\nbuild/\n.packages\n", true}
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

// existsGlob reports whether any file in the current folder matches pattern.
func existsGlob(pattern string) bool {
	m, err := filepath.Glob(pattern)
	return err == nil && len(m) > 0
}
