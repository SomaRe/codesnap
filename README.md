CodeSnap
========

CodeSnap is a utility that helps you quickly capture and share code structure from your projects. Available in both Go and Python implementations, it allows you to specify folders and files to include, with support for ignore patterns, and copies the content to your clipboard in a well-formatted manner.

***Note:** Future versions will not include the Python implementation.*

Features
--------

-   📁 Capture content from multiple folders and files
-   🚫 Ignore specific patterns (like test files or node_modules)
-   📋 Automatic clipboard copying
-   🔧 YAML configuration
-   ⏱️ Performance metrics (execution time)

Go Implementation
-----------------

### Installation

1.  Download the latest `codesnap.exe` from releases
2.  Add to PATH:
    -   Press Win + X and select "System"
    -   Click "Advanced system settings"
    -   Click "Environment Variables"
    -   Under "System Variables", find and select "Path"
    -   Click "New" and add the directory containing codesnap.exe

### Building from Source (from windows)

```bash
go build -o codesnap.exe main.go
```

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o dist/codesnap-amd64-linux main.go
```

Python Implementation
---------------------

### Installation

#### Windows

1.  Download `install.bat`
2.  Double-click to install
    -   Downloads latest version
    -   Adds to PATH
    -   Installs required packages

Usage (Both Implementations)
----------------------------

1.  Navigate to your project directory:

```bash
cd your-project
```

2.  Run CodeSnap:

```bash
codesnap
```

3.  First run creates a template `codesnap.yml`:

```yaml
folders:
  - src           # relative to config file
  - utils         # project subdirectory

files:
  - package.json  # individual files
  - config.js

ignore:
  - "**/*.test.js"    # ignore test files
  - "**/node_modules/**"
  - "**/.git/**"
```

4.  Edit the configuration and run again to copy content to clipboard

Command Line Arguments
----------------------

```bash
codesnap [-h] [-c CONFIG] [-p] [-o] [-v]
```

-   `-h, --help`: Show help message
-   `-c, --config`: Specify config file path
-   `-p, --print`: Print to terminal
-   `-o, --output`: Save to file
-   `-v, --version`: Show version

Performance comparison code results
----------------------------------

**summary:** 

```bash
Running performance comparison (5 runs each)...

Building Go executable...
Go build completed in 0.323 seconds.

Testing Go Executable, Go Run, and Python Run...

Run 1/5
Running Go executable...
Running Go program (using `go run`)...
Running Python script...

Run 2/5
Running Go executable...
Running Go program (using `go run`)...
Running Python script...

Run 3/5
Running Go executable...
Running Go program (using `go run`)...
Running Python script...

============================================================
Performance Comparison Results
============================================================

GO EXECUTABLE PERFORMANCE:
----------------------------------------
  Average run time: 1.943 seconds
  Standard deviation: 0.062 seconds

GO RUN PERFORMANCE (using `go run`):
----------------------------------------
  Average run time: 2.533 seconds
  Standard deviation: 0.032 seconds

PYTHON RUN PERFORMANCE:
----------------------------------------
  Average run time: 3.665 seconds
  Standard deviation: 0.075 seconds

RELATIVE PERFORMANCE COMPARISON:
----------------------------------------
`go executable` is 1.30x faster than `go run`
`go executable` is 1.89x faster than `python`
`go run` is 1.30x slower than `go executable`
`python` is 1.89x slower than `go executable`
`python` is 1.45x slower than `go run`
```

Future Plans
------------

### Go Implementation

-   Auto-update mechanism
-   Simplified installation process
-   Cross-platform installers
-   PATH setup automation

### Python Implementation

-   Package manager distribution
-   Enhanced installation methods
-   IDE integrations

Contributing
------------

Contributions are welcome! Currently looking for help with:

-   MacOS and Linux support
-   Installation automation
-   Cross-platform testing

License
-------

MIT License - see LICENSE file

Author
------

SomaRe (<https://github.com/SomaRe>)