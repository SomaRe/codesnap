#!/usr/bin/python
import os
import glob
import pyperclip
import yaml
import argparse
from pathlib import Path
import sys
from datetime import datetime
import time

class Logger:
    def __init__(self, log_to_file=False):
        self.log_to_file = log_to_file
        if self.log_to_file:
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
            self.log_file = f"codesnap_log_{timestamp}.txt"
        self.messages = []
        
    def log(self, message, message_type="INFO"):
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        log_entry = f"[{timestamp}] {message_type}: {message}"
        self.messages.append(log_entry)
        
        # Write to file only if logging is enabled
        if self.log_to_file:
            with open(self.log_file, 'a', encoding='utf-8') as f:
                f.write(log_entry + '\n')
    
    def warning(self, message):
        self.log(message, "WARNING")
    
    def error(self, message):
        self.log(message, "ERROR")
    
    def info(self, message):
        self.log(message, "INFO")

# Create a global logger instance
logger = Logger()

def print_help():
    """Print help message with examples"""
    help_text = """
CodeSnap - Copy your code structure to clipboard

Usage: 
    codesnap [options]

Options:
    -h, --help          Show this help message
    -c, --config PATH   Specify path to config file (default: codesnap.yml in current directory)
    -p, --print         Print the collected content to terminal
    -o, --output        Save content to a timestamped text file
    -v, --version       Show version number

Examples:
    # Run with default config file (codesnap.yml in current directory)
    codesnap

    # Use specific config file
    codesnap -c /path/to/config.yml

    # Print content to terminal
    codesnap -p

    # Save to text file
    codesnap -o

    # Print to terminal and save to file
    codesnap -p -o

Config File Format (codesnap.yml):
    folders:            # Folders to include (relative to config file)
    -l, --log           # Save log of processed files to a log file
    -v, --version       # Show version number

    ignore:            # Patterns to ignore
        - "**/*.test.js"
        - "**/node_modules/**"
        - "**/.git/**"
    """
    logger.info(help_text)

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
            logger.info("No codesnap.yml found. Creating template configuration file...")
            try:
                with open(config_file, 'w', encoding='utf-8') as f:
                    f.write(TEMPLATE_CONFIG)
                logger.info(f"Created template configuration at: {config_file}")
                logger.info("Please edit the file and run codesnap again.")
                sys.exit(0)
            except Exception as e:
                raise CodeSnapError(f"Failed to create template configuration: {str(e)}")
                
        return config_file

    def _load_config(self):
        """Load configuration from YAML file."""
        try:
            with open(self.config_path, 'r', encoding='utf-8') as f:
                config = yaml.safe_load(f) or {}

            # Initialize empty lists for missing sections
            config['folders'] = [] if config.get('folders') is None else config['folders']
            config['files'] = [] if config.get('files') is None else config['files']
            config['ignore'] = [] if config.get('ignore') is None else config['ignore']
                
            if not isinstance(config['folders'], list):
                raise CodeSnapError("'folders' must be a list")
            if not isinstance(config['files'], list):
                raise CodeSnapError("'files' must be a list")
            if not isinstance(config['ignore'], list):
                raise CodeSnapError("'ignore' must be a list")
            
            if not config['folders'] and not config['files']:
                raise CodeSnapError("Configuration must specify at least one file or folder to process")
                
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
            logger.warning(f"Skipping binary file: {file_path}")
            return ""
        except Exception as e:
            logger.warning(f"Could not read {file_path}: {str(e)}")
            return ""

    def should_include_file(self, file_path):
        """Check if file should be included based on ignore patterns."""
        try:
            ignore_patterns = self.config.get('ignore', [])
            rel_path = Path(file_path).relative_to(self.config_path.parent)
            
            return not any(glob.fnmatch.fnmatch(str(rel_path), pattern) 
                         for pattern in ignore_patterns)
        except Exception:
            return True

    def collect_content(self):
        """Collect content from all specified files and folders."""
        all_content = []
        processed_files = 0
        
        # Process folders
        for folder in self.config['folders']:
            folder_path = self._resolve_path(folder)
            if not os.path.exists(folder_path):
                logger.warning(f"Folder not found: {folder}")
                continue
                
            folder_files = 0
            for file_path in glob.glob(os.path.join(folder_path, '**'), recursive=True):
                if os.path.isfile(file_path) and self.should_include_file(file_path):
                    content = self.get_file_content(file_path)
                    if content:
                        rel_path = Path(file_path).relative_to(self.config_path.parent)
                        all_content.append(f"\n\n{'='*50}\nFile: {rel_path}\n{'='*50}\n\n{content}")
                        folder_files += 1
            
            processed_files += folder_files
            if folder_files == 0:
                logger.warning(f"No valid files found in folder: {folder}")

        # Process individual files
        for file_path in self.config['files']:
            resolved_path = self._resolve_path(file_path)
            if not os.path.exists(resolved_path):
                logger.warning(f"File not found: {file_path}")
                continue
                
            if self.should_include_file(resolved_path):
                content = self.get_file_content(resolved_path)
                if content:
                    rel_path = Path(resolved_path).relative_to(self.config_path.parent)
                    all_content.append(f"\n\n{'='*50}\nFile: {rel_path}\n{'='*50}\n\n{content}")
                    processed_files += 1

        if processed_files == 0:
            raise CodeSnapError("No valid content found to copy. Please check your configuration and ensure files exist.")
            
        return "\n".join(all_content)

    def save_to_file(self, content):
        """Save content to a text file with timestamp."""
        timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
        filename = f"codesnap_{timestamp}.txt"
        
        try:
            with open(filename, 'w', encoding='utf-8') as f:
                f.write(content)
            logger.info(f"Content saved to: {filename}")
        except Exception as e:
            raise CodeSnapError(f"Failed to save content to file: {str(e)}")

def main():
    start_time = time.time()

    parser = argparse.ArgumentParser(
        description='Copy code structure to clipboard. Creates a template configuration if none exists.',
        formatter_class=argparse.RawTextHelpFormatter
    )
    parser.add_argument('-c', '--config', help='Path to config file (default: codesnap.yml in current directory)')
    parser.add_argument('-p', '--print', action='store_true', help='Print the collected content to terminal')
    parser.add_argument('-o', '--output', action='store_true', help='Save the content to a text file')
    parser.add_argument('-l', '--log', action='store_true', help='Enable logging to a file')
    parser.add_argument('-v', '--version', action='store_true', help='Show version number')
    parser.add_argument('--help-extended', action='store_true', help='Show extended help with examples')

    args = parser.parse_args()

    if args.help_extended:
        print_help()
        sys.exit(0)

    if args.version:
        print("CodeSnap version 1.0.0")
        sys.exit(0)

    # Initialize logger with the log flag
    global logger
    logger = Logger(log_to_file=args.log)

    try:
        snapper = CodeSnap(args.config)
        final_content = snapper.collect_content()
        pyperclip.copy(final_content)
        logger.info("Successfully copied content to clipboard!")
        print("Content copied to clipboard!")
        
        if args.print:
            logger.info(f"Processed files are:\n{final_content}")
            
        if args.output:
            snapper.save_to_file(final_content)
            
    except CodeSnapError as e:
        logger.error(str(e))
        logger.error("For help, use --help-extended to see examples and configuration format")
        sys.exit(1)
    except KeyboardInterrupt:
        logger.error("Operation cancelled by user.")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}")
        logger.error("Please report this issue if it persists.")
        sys.exit(1)

    elapsed_time = time.time() - start_time
    logger.info(f"Total execution time: {elapsed_time:.3f} seconds")
    print(f"Total execution time: {elapsed_time:.3f} seconds")


if __name__ == "__main__":
    main()