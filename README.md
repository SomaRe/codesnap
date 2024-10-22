# CodeSnap

CodeSnap is a command-line utility that helps you quickly capture and share code structure from your projects. It allows you to specify folders and files to include, with support for ignore patterns, and copies the content to your clipboard in a well-formatted manner.

## Features

- 📁 Capture content from multiple folders and files
- 🚫 Ignore specific patterns (like test files or node_modules)
- 📋 Automatic clipboard copying
- 🔧 YAML configuration with examples
- 💾 Creates template configuration if none exists
- 🌟 Support for relative and absolute paths

## Installation

### Windows
1. Download `install.bat`
2. Double-click to install
   - Automatically downloads latest version from GitHub
   - Falls back to local file if download fails
   - Adds to PATH
   - Installs required Python packages

### macOS (Coming Soon)
Planned installation methods:
```bash
# Via Homebrew (coming soon)
brew install somare/tools/codesnap

# Via curl installer (coming soon)
curl -sSL https://raw.githubusercontent.com/SomaRe/codesnap/main/install.sh | bash
```

### Linux (Coming Soon)
Planned installation methods:
```bash
# Debian/Ubuntu (coming soon)
apt-get install codesnap

# Via wget installer (coming soon)
wget -qO- https://raw.githubusercontent.com/SomaRe/codesnap/main/install.sh | bash
```

## Usage

1. Navigate to your project directory:
```bash
cd your-project
```

2. Run CodeSnap:
```bash
codesnap
```

3. If no configuration exists, it will create a template `codesnap.yml` file:
```yaml
# CodeSnap Configuration File
folders:
  - src           # relative to this config file
  - ../shared     # parent directory
  - utils         # project subdirectory

files:
  - package.json  # individual files to include
  - config.js     # relative to this config file

ignore:
  - "**/*.test.js"    # ignore test files
  - "**/node_modules/**"  # ignore node_modules
  - "**/.git/**"     # ignore git directory
```

4. Edit the configuration file to specify which folders and files to include
5. Run `codesnap` again to copy the content to your clipboard

## Configuration

### File Location
The configuration file should be named `codesnap.yml` and placed in your project root directory.

### Structure
- `folders`: List of folders to include (recursive)
- `files`: List of individual files to include
- `ignore`: List of glob patterns to exclude

### Path Support
- Relative paths (relative to config file location)
- Absolute paths
- Project subdirectories

## Future Plans

1. Package Manager Distribution
   - Homebrew formula for macOS
   - APT package for Debian/Ubuntu
   - RPM package for Red Hat/Fedora

2. Enhanced Installation Methods
   - Shell script installer for Unix systems
   - DMG installer for macOS
   - MSI installer for Windows

3. Additional Features
   - Output formatting options
   - Custom templates
   - Direct sharing capabilities
   - IDE integrations

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request, definetly need help with MacOS and Linux.

## License

MIT License - see the LICENSE file for details

## Author

- SomaRe (https://github.com/SomaRe)