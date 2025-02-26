package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v2"
)

const version = "1.1.0"
const templateConfig = `# CodeSnap Configuration File
# Examples:
# folders:
# - src # relative to this config file
# - ../shared # parent directory
# - utils # project subdirectory
# files:
# - package.json # individual files to include
# - config.js # relative to this config file
# ignore:
# - "**/*.test.js" # ignore test files
# - "**/node_modules/**" # ignore node_modules
# - "**/.git/**" # ignore git directory
# - "**/*.jpg" # ignore image files
# - "**/*.png" # ignore image files
# - "**/*.gif" # ignore image files
# - "**/*.pdf" # ignore PDF files
# - "**/*.exe" # ignore executable files
# - "**/*.dll" # ignore DLL files
# tree_depth: 3 # maximum depth for folder structure (default: unlimited)
folders:
files:
ignore:
tree_depth:  
`

type Config struct {
	Folders   []string `yaml:"folders"`
	Files     []string `yaml:"files"`
	Ignore    []string `yaml:"ignore"`
	TreeDepth int      `yaml:"tree_depth"`
}

type CodeSnap struct {
	configPath string
	config     *Config
	baseDir    string
}

// FileResult represents the processing result of a single file
type FileResult struct {
	path    string
	relPath string
	content string
	isEmpty bool
	err     error
	isValid bool
	skipped bool
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

// findOrCreateConfig searches for a configuration file at the specified path
// and creates one if not found. If a file is created, the program exits with
// code 0 after printing instructions to the user.
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
	// First get the config file's directory
	configDir := filepath.Dir(cs.configPath)
	// Then join it with the relative path
	return filepath.Join(configDir, path)
}

func (cs *CodeSnap) shouldIncludeFile(path string) bool {
	// Convert the file path to forward slashes
	relPath, err := filepath.Rel(filepath.Dir(cs.configPath), path)
	if err != nil {
		return true
	}

	// Convert to forward slashes for consistent matching
	relPath = filepath.ToSlash(relPath)
	for _, pattern := range cs.config.Ignore {
		// Convert backslashes to forward slashes in the pattern
		pattern = filepath.ToSlash(pattern)
		matched, err := doublestar.Match(pattern, relPath)
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

// New structure for tracking stats atomically
type Stats struct {
	processed int64
	empty     int64
	skipped   int64
}

func (cs *CodeSnap) collectContent(logOutput bool) (string, error) {
	var outputFile string
	if logOutput {
		outputFile = fmt.Sprintf("codesnap_log_%s.txt", time.Now().Format("20060102_150405"))
	}

	var stats Stats
	var filePaths []string
	var mutex sync.Mutex
	var wg sync.WaitGroup
	var resultsMutex sync.Mutex
	var results []*FileResult
	// var errOccurred atomic.Bool

	// First build the list of files to process
	// Process configured folders
	for _, folder := range cs.config.Folders {
		folderPath := cs.resolvePath(folder)
		// Check if folder exists
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			if logOutput {
				saveToOutput(fmt.Sprintf("Folder not found: %s", folderPath), outputFile)
			}
			continue
		}

		fmt.Printf("Finding files in folder: %s\n", folderPath)
		// Create pattern for all files in the folder
		pattern := filepath.Join(folderPath, "**")
		// Use FilepathGlob to find all matching files
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			if logOutput {
				saveToOutput(fmt.Sprintf("Error globbing folder %s: %v", folderPath, err), outputFile)
			}
			continue
		}

		// Add matched files to the list
		for _, match := range matches {
			// Skip directories
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			if cs.shouldIncludeFile(match) {
				filePaths = append(filePaths, match)
			}
		}
	}

	// Process individual files
	for _, file := range cs.config.Files {
		filePath := cs.resolvePath(file)
		if cs.shouldIncludeFile(filePath) {
			filePaths = append(filePaths, filePath)
		}
	}

	fmt.Printf("Found %d files to process\n", len(filePaths))

	// Create a channel to communicate work to worker goroutines
	jobs := make(chan string, len(filePaths))
	results = make([]*FileResult, 0, len(filePaths))

	// Determine the number of workers based on CPU cores
	numWorkers := 8 // You can adjust this based on your machine

	// Create worker goroutines
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for filePath := range jobs {
				relPath, _ := filepath.Rel(filepath.Dir(cs.configPath), filePath)
				isValid, content, err := validateFile(filePath)

				result := &FileResult{
					path:    filePath,
					relPath: relPath,
					isValid: isValid,
				}

				if err != nil {
					atomic.AddInt64(&stats.skipped, 1)
					result.err = err
					result.skipped = true
					if logOutput {
						saveToOutput(fmt.Sprintf("Skipping %s: %v", relPath, err), outputFile)
					}
				} else {
					atomic.AddInt64(&stats.processed, 1)
					result.content = content
					if !isValid || len(content) == 0 {
						atomic.AddInt64(&stats.empty, 1)
						result.isEmpty = true
					}
				}

				resultsMutex.Lock()
				results = append(results, result)
				resultsMutex.Unlock()

				processed := atomic.LoadInt64(&stats.processed)
				skipped := atomic.LoadInt64(&stats.skipped)
				total := processed + skipped

				if total%50 == 0 || total == int64(len(filePaths)) {
					fmt.Printf("\rProcessed: %d, Skipped: %d, Total: %d/%d",
						processed, skipped, total, len(filePaths))
				}
			}
		}(w)
	}

	// Send work to workers
	for _, filePath := range filePaths {
		jobs <- filePath
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	fmt.Println() // Add a newline after progress

	// Sort results to maintain consistent order
	mutex.Lock()
	defer mutex.Unlock()

	// Build the final content string
	var allContent strings.Builder

	for _, result := range results {
		if result.skipped {
			continue
		}

		if result.isEmpty {
			allContent.WriteString(fmt.Sprintf("\n\n%s\nFile: %s (empty)\n%s",
				strings.Repeat("=", 50), result.relPath, strings.Repeat("=", 50)))
		} else {
			allContent.WriteString(fmt.Sprintf("\n\n%s\nFile: %s\n%s\n\n%s",
				strings.Repeat("=", 50), result.relPath, strings.Repeat("=", 50), result.content))
		}
	}

	// Add summary
	processed := atomic.LoadInt64(&stats.processed)
	empty := atomic.LoadInt64(&stats.empty)
	skipped := atomic.LoadInt64(&stats.skipped)

	summary := fmt.Sprintf("\n\n%s\nSummary:\n"+
		"- Files processed: %d\n"+
		"- Empty files: %d\n"+
		"- Files skipped: %d\n%s",
		strings.Repeat("=", 50),
		processed,
		empty,
		skipped,
		strings.Repeat("=", 50))
	allContent.WriteString(summary)

	if processed == 0 {
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

func (cs *CodeSnap) generateFolderStructure() (string, error) {
	var buffer strings.Builder
	var stats struct {
		dirs  int
		files int
	}

	// Helper function to print the tree structure
	var printTree func(path string, prefix string, isLast bool, depth int) error
	printTree = func(path string, prefix string, isLast bool, depth int) error {
		if cs.config.TreeDepth > 0 && depth > cs.config.TreeDepth {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		// Create the current line prefix
		currentPrefix := prefix
		if isLast {
			currentPrefix += "└── "
		} else {
			currentPrefix += "├── "
		}
		// Add the current item to the output
		if info.IsDir() {
			buffer.WriteString(fmt.Sprintf("%s%s/\n", currentPrefix, filepath.Base(path)))
			stats.dirs++
		} else {
			buffer.WriteString(fmt.Sprintf("%s%s\n", currentPrefix, filepath.Base(path)))
			stats.files++
		}
		// If it's a directory, process its contents
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			// Filter and sort entries
			var filteredEntries []os.DirEntry
			for _, entry := range entries {
				fullPath := filepath.Join(path, entry.Name())
				if cs.shouldIncludeFile(fullPath) {
					filteredEntries = append(filteredEntries, entry)
				}
			}
			for i, entry := range filteredEntries {
				isLastEntry := i == len(filteredEntries)-1
				nextPrefix := prefix
				if isLast {
					nextPrefix += "    "
				} else {
					nextPrefix += "│   "
				}
				err := printTree(filepath.Join(path, entry.Name()), nextPrefix, isLastEntry, depth+1)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	// Process configured folders
	for i, folder := range cs.config.Folders {
		folderPath := cs.resolvePath(folder)
		buffer.WriteString(fmt.Sprintf("Folder: %s\n", folder))
		if err := printTree(folderPath, "", i == len(cs.config.Folders)-1, 0); err != nil {
			return "", fmt.Errorf("error processing folder %s: %v", folder, err)
		}
		buffer.WriteString("\n")
	}
	// Add summary
	summary := fmt.Sprintf("\nStructure Summary:\n"+
		"- Directories: %d\n"+
		"- Files: %d\n",
		stats.dirs,
		stats.files)
	buffer.WriteString(summary)
	if stats.dirs == 0 && stats.files == 0 {
		return "", fmt.Errorf("no valid folders or files were found")
	}
	return buffer.String(), nil
}

func printHelp() {
	helpText := `
CodeSnap - Copy your code structure to clipboard
Usage:
  codesnap [options]

Options:
  -h, --help       Show this help message
  -c, --config PATH Specify path to config file (default: codesnap.yml in current directory)
  -p, --print      Print the collected content to terminal
  -o, --output     Save content to a timestamped text file
  -l, --log        Save log of processed files to a log file
  -t, --tree       Generate and copy folder structure tree
  -v, --version    Show version number
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
	showTree := flag.Bool("t", false, "Generate and copy folder structure tree")

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

	var content string
	if *showTree {
		content, err = cs.generateFolderStructure()
	} else {
		content, err = cs.collectContent(*logOutput)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := clipboard.WriteAll(content); err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
		os.Exit(1)
	}

	if *showTree {
		fmt.Println("\nDirectory tree structure successfully copied to clipboard!")
	} else {
		fmt.Println("\nSuccessfully copied code content to clipboard!")
	}

	if *printContent {
		fmt.Printf("\nContent:\n%s\n", content)
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
