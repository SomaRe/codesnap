#!/usr/bin/env python3
import os
import glob
import pyperclip
import yaml
import argparse
from pathlib import Path
import sys

TEMPLATE_CONFIG = '''# CodeSnap Configuration File
# Examples:
# folders:
#   - src           # relative to this config file
#   - ../shared     # parent directory
#   - utils         # project subdirectory
#
# files:
#   - package.json  # individual files to include
#   - config.js     # relative to this config file
#
# ignore:
#   - "**/*.test.js"    # ignore test files
#   - "**/node_modules/**"  # ignore node_modules
#   - "**/.git/**"     # ignore git directory

folders:

files:

ignore:
'''

class CodeSnapError(Exception):
    """Custom exception for CodeSnap errors"""
    pass

class CodeSnap:
    def __init__(self, config_path=None):
        self.base_dir = Path.cwd()
        try:
            self.config_path = config_path or self._find_or_create_config()
            self.config = self._load_config()
        except Exception as e:
            raise CodeSnapError(f"Configuration error: {str(e)}")

    def _find_or_create_config(self):
        """Search for codesnap.yml or create if not found."""
        current = Path.cwd()
        config_file = current / 'codesnap.yml'
        
        if not config_file.exists():
            print("No codesnap.yml found. Creating template configuration file...")
            try:
                with open(config_file, 'w', encoding='utf-8') as f:
                    f.write(TEMPLATE_CONFIG)
                print(f"Created template configuration at: {config_file}")
                print("Please edit the file and run codesnap again.")
                sys.exit(0)
            except Exception as e:
                raise CodeSnapError(f"Failed to create template configuration: {str(e)}")
                
        return config_file

    def _load_config(self):
        """Load configuration from YAML file."""
        try:
            with open(self.config_path, 'r', encoding='utf-8') as f:
                config = yaml.safe_load(f) or {}
                
            # Validate configuration structure
            if not isinstance(config.get('folders', []), list):
                raise CodeSnapError("'folders' must be a list")
            if not isinstance(config.get('files', []), list):
                raise CodeSnapError("'files' must be a list")
            if not isinstance(config.get('ignore', []), list):
                raise CodeSnapError("'ignore' must be a list")
                
            return config
        except yaml.YAMLError as e:
            raise CodeSnapError(f"Invalid YAML format: {str(e)}")
        except Exception as e:
            raise CodeSnapError(f"Failed to load configuration: {str(e)}")

    def _resolve_path(self, path):
        """Resolve relative paths to absolute paths."""
        try:
            path = Path(path)
            if not path.is_absolute():
                return str(self.config_path.parent / path)
            return str(path)
        except Exception as e:
            raise CodeSnapError(f"Invalid path format: {path}")

    def get_file_content(self, file_path):
        """Read content from a file with error handling."""
        try:
            with open(file_path, 'r', encoding='utf-8') as file:
                return file.read()
        except UnicodeDecodeError:
            print(f"Warning: Skipping binary file: {file_path}")
            return ""
        except Exception as e:
            print(f"Warning: Could not read {file_path}: {str(e)}")
            return ""

    def should_include_file(self, file_path):
        """Check if file should be included based on ignore patterns."""
        try:
            ignore_patterns = self.config.get('ignore', [])
            rel_path = Path(file_path).relative_to(self.config_path.parent)
            
            return not any(glob.fnmatch.fnmatch(str(rel_path), pattern) 
                         for pattern in ignore_patterns)
        except Exception:
            return True  # If pattern matching fails, include the file

    def collect_content(self):
        """Collect content from all specified files and folders."""
        all_content = []
        folders = self.config.get('folders', [])
        files = self.config.get('files', [])
        
        if not folders and not files:
            raise CodeSnapError("No folders or files specified in configuration")

        # Get content from folders
        for folder in folders:
            folder_path = self._resolve_path(folder)
            if not os.path.exists(folder_path):
                print(f"Warning: Folder not found: {folder}")
                continue
                
            for file_path in glob.glob(os.path.join(folder_path, '**'), recursive=True):
                if os.path.isfile(file_path) and self.should_include_file(file_path):
                    content = self.get_file_content(file_path)
                    if content:  # Only add non-empty content
                        rel_path = Path(file_path).relative_to(self.config_path.parent)
                        all_content.append(f"\n\n{'='*50}\nFile: {rel_path}\n{'='*50}\n\n{content}")

        # Get content from specific files
        for file_path in files:
            resolved_path = self._resolve_path(file_path)
            if not os.path.exists(resolved_path):
                print(f"Warning: File not found: {file_path}")
                continue
                
            if self.should_include_file(resolved_path):
                content = self.get_file_content(resolved_path)
                if content:  # Only add non-empty content
                    rel_path = Path(resolved_path).relative_to(self.config_path.parent)
                    all_content.append(f"\n\n{'='*50}\nFile: {rel_path}\n{'='*50}\n\n{content}")

        if not all_content:
            raise CodeSnapError("No valid content found to copy")
            
        return "\n".join(all_content)

def main():
    parser = argparse.ArgumentParser(
        description='Copy code structure to clipboard. Creates a template configuration if none exists.'
    )
    parser.add_argument('-c', '--config', help='Path to config file (default: codesnap.yml in current directory)')
    parser.add_argument('-p', '--print', action='store_true', help='Print the collected content to terminal')
    args = parser.parse_args()

    try:
        snapper = CodeSnap(args.config)
        final_content = snapper.collect_content()
        pyperclip.copy(final_content)
        print("\nSuccessfully copied content to clipboard!")
        if args.print:
            print(f"\nProcessed files are:\n{final_content}")
    except CodeSnapError as e:
        print(f"\nError: {str(e)}")
        print("\nFor help, ensure your codesnap.yml is properly configured.")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nOperation cancelled by user.")
        sys.exit(1)
    except Exception as e:
        print(f"\nUnexpected error: {str(e)}")
        print("Please report this issue if it persists.")
        sys.exit(1)

if __name__ == "__main__":
    main()