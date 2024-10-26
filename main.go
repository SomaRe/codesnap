package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"gopkg.in/yaml.v2"
)

const version = "1.0.0"
const templateConfig = `# CodeSnap Configuration File
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
#   - "**/*.jpg"       # ignore image files
#   - "**/*.png"       # ignore image files
#   - "**/*.gif"       # ignore image files
#   - "**/*.pdf"       # ignore PDF files
#   - "**/*.exe"       # ignore executable files
#   - "**/*.dll"       # ignore DLL files

folders:

files:

ignore:
`

type Config struct {
	Folders []string `yaml:"folders"`
	Files   []string `yaml:"files"`
	Ignore  []string `yaml:"ignore"`
}

type CodeSnap struct {
	configPath string
	config     *Config
	baseDir    string
}

// validateFile checks if a file is a readable text file
func validateFile(filepath string) (bool, string, error) {
	// Check if file exists and is readable
	file, err := os.Open(filepath)
	if err != nil {
		return false, "", fmt.Errorf("cannot open file: %v", err)
	}
	defer file.Close()

	// Check file size
	info, err := file.Stat()
	if err != nil {
		return false, "", fmt.Errorf("cannot stat file: %v", err)
	}

	// Handle empty files
	if info.Size() == 0 {
		return true, "", nil // Empty files are valid but have no content
	}

	// Read first 8KB of the file
	// This is usually enough to detect if it's text while not reading entire large files
	buf := make([]byte, 8*1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, "", fmt.Errorf("error reading file: %v", err)
	}
	buf = buf[:n]

	// Check for null bytes (common in binary files)
	if bytes.Contains(buf, []byte{0}) {
		return false, "", fmt.Errorf("file appears to be binary (contains null bytes)")
	}

	// Check if content is valid UTF-8
	if !utf8.Valid(buf) {
		return false, "", fmt.Errorf("file contains invalid UTF-8 characters")
	}

	// Read the actual content if validation passed
	file.Seek(0, 0) // Reset to beginning of file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return false, "", fmt.Errorf("error reading full file content: %v", err)
	}

	return true, string(content), nil
}

func NewCodeSnap(configPath string) (*CodeSnap, error) {
	if configPath == "" {
		configPath = "codesnap.yml"
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %v", err)
	}

	cs := &CodeSnap{
		configPath: configPath,
		baseDir:    baseDir,
	}

	if err := cs.findOrCreateConfig(); err != nil {
		return nil, err
	}

	if err := cs.loadConfig(); err != nil {
		return nil, err
	}

	return cs, nil
}

func (cs *CodeSnap) findOrCreateConfig() error {
	if _, err := os.Stat(cs.configPath); os.IsNotExist(err) {
		fmt.Println("No codesnap.yml found. Creating template configuration file...")
		if err := os.WriteFile(cs.configPath, []byte(templateConfig), 0644); err != nil {
			return fmt.Errorf("failed to create template configuration: %v", err)
		}
		fmt.Printf("Created template configuration at: %s\n", cs.configPath)
		fmt.Println("Please edit the file and run codesnap again.")
		os.Exit(0)
	}
	return nil
}

func (cs *CodeSnap) loadConfig() error {
	data, err := os.ReadFile(cs.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	cs.config = &Config{}
	if err := yaml.Unmarshal(data, cs.config); err != nil {
		return fmt.Errorf("invalid YAML format: %v", err)
	}

	// Initialize empty slices if they're nil
	if cs.config.Folders == nil {
		cs.config.Folders = []string{}
	}
	if cs.config.Files == nil {
		cs.config.Files = []string{}
	}
	if cs.config.Ignore == nil {
		cs.config.Ignore = []string{}
	}

	if len(cs.config.Folders) == 0 && len(cs.config.Files) == 0 {
		return fmt.Errorf("configuration must specify at least one file or folder to process")
	}

	return nil
}

func (cs *CodeSnap) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(cs.configPath), path)
}

func (cs *CodeSnap) shouldIncludeFile(path string) bool {
	relPath, err := filepath.Rel(filepath.Dir(cs.configPath), path)
	if err != nil {
		return true
	}

	for _, pattern := range cs.config.Ignore {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return false
		}
	}
	return true
}

// Add this function for saving output
func saveToOutput(message string, outputFile string) error {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Open output file in append mode, create if doesn't exist
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}

	return nil
}

func (cs *CodeSnap) collectContent(logOutput bool) (string, error) {
	var outputFile string
	if logOutput {
		outputFile = fmt.Sprintf("codesnap_log_%s.txt", time.Now().Format("20060102_150405"))
	}

	var allContent strings.Builder
	var stats struct {
		processed int
		empty     int
		skipped   int
	}

	// Helper function to process a single file
	processFile := func(path string) {
		relPath, _ := filepath.Rel(filepath.Dir(cs.configPath), path)

		isValid, content, err := validateFile(path)
		if err != nil {
			stats.skipped++
			if logOutput {
				saveToOutput(fmt.Sprintf("Skipping %s: %v", relPath, err), outputFile)
			}
			return
		}

		stats.processed++
		if !isValid || len(content) == 0 {
			stats.empty++
			allContent.WriteString(fmt.Sprintf("\n\n%s\nFile: %s (empty)\n%s",
				strings.Repeat("=", 50), relPath, strings.Repeat("=", 50)))
		} else {
			allContent.WriteString(fmt.Sprintf("\n\n%s\nFile: %s\n%s\n\n%s",
				strings.Repeat("=", 50), relPath, strings.Repeat("=", 50), content))
		}
	}

	// Process configured folders
	for _, folder := range cs.config.Folders {
		folderPath := cs.resolvePath(folder)
		filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if logOutput {
					saveToOutput(fmt.Sprintf("Error accessing %s: %v", path, err), outputFile)
				}
				return nil
			}

			if !info.IsDir() && cs.shouldIncludeFile(path) {
				processFile(path)
			}
			return nil
		})
	}

	// Process individual files
	for _, file := range cs.config.Files {
		filePath := cs.resolvePath(file)
		if cs.shouldIncludeFile(filePath) {
			processFile(filePath)
		}
	}

	// Add summary
	summary := fmt.Sprintf("\n\n%s\nSummary:\n"+
		"- Files processed: %d\n"+
		"- Empty files: %d\n"+
		"- Files skipped: %d\n%s",
		strings.Repeat("=", 50),
		stats.processed,
		stats.empty,
		stats.skipped,
		strings.Repeat("=", 50))
	allContent.WriteString(summary)

	if stats.processed == 0 {
		return "", fmt.Errorf("no valid files were processed")
	}

	return allContent.String(), nil
}

func (cs *CodeSnap) saveToFile(content string) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("codesnap_%s.txt", timestamp)

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save content to file: %v", err)
	}

	fmt.Printf("Content saved to: %s\n", filename)
	return nil
}

func printHelp() {
	helpText := `
CodeSnap - Copy your code structure to clipboard

Usage: 
    codesnap [options]

Options:
    -h, --help          Show this help message
    -c, --config PATH   Specify path to config file (default: codesnap.yml in current directory)
    -p, --print         Print the collected content to terminal
    -o, --output        Save content to a timestamped text file
    -l, --log           Save log of processed files to a log file
    -v, --version       Show version number
`
	fmt.Println(helpText)
}

func main() {
	startTime := time.Now()

	configPath := flag.String("c", "", "Path to config file")
	printContent := flag.Bool("p", false, "Print the collected content to terminal")
	saveOutput := flag.Bool("o", false, "Save the content to a text file")
	logOutput := flag.Bool("l", false, "Save log of processed files to a log file")
	showVersion := flag.Bool("v", false, "Show version number")
	showHelp := flag.Bool("h", false, "Show help message")

	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Printf("CodeSnap version %s\n", version)
		return
	}

	cs, err := NewCodeSnap(*configPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	content, err := cs.collectContent(*logOutput)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := clipboard.WriteAll(content); err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nSuccessfully copied content to clipboard!")

	if *printContent {
		fmt.Printf("\nProcessed files are:\n%s\n", content)
	}

	if *saveOutput {
		if err := cs.saveToFile(content); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nTotal execution time: %v\n", elapsed)
}
