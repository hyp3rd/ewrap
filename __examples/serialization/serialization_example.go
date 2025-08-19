package main

import (
	"fmt"
	"log"

	"github.com/goccy/go-json"

	"github.com/hyp3rd/ewrap"
)

func main() {
	// Example 1: ErrorGroup serialization
	fmt.Println("=== ErrorGroup Serialization Example ===")

	eg := ewrap.NewErrorGroup()

	// Add various types of errors
	eg.Add(ewrap.New("first error").WithMetadata("key1", "value1"))
	eg.Add(ewrap.Wrap(fmt.Errorf("standard error"), "wrapped standard error"))
	eg.Add(ewrap.New("another error").WithMetadata("severity", "high"))

	// Serialize to JSON
	jsonData, err := eg.ToJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("JSON serialization:\n%s\n\n", string(jsonData))

	// Serialize to YAML
	yamlData, err := eg.ToYAML()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("YAML serialization:\n%s\n\n", string(yamlData))

	// Example 2: Stack frame inspection
	fmt.Println("=== Stack Frame Inspection Example ===")

	err1 := createErrorInFunction()

	// Get stack iterator
	iterator := err1.GetStackIterator()

	fmt.Println("Stack frames:")
	frameNum := 0
	for iterator.HasNext() {
		frame := iterator.Next()
		fmt.Printf("Frame %d: %s:%d in %s\n",
			frameNum, frame.File, frame.Line, frame.Function)
		frameNum++

		// Limit output for readability
		if frameNum >= 5 {
			break
		}
	}

	// Example 3: Get all stack frames as slice
	fmt.Println("\n=== All Stack Frames ===")
	frames := err1.GetStackFrames()
	for i, frame := range frames {
		fmt.Printf("Frame %d: %s:%d in %s\n",
			i, frame.File, frame.Line, frame.Function)
		if i >= 3 { // Limit for readability
			break
		}
	}

	// Example 4: Structured error representation
	fmt.Println("\n=== Structured Error Representation ===")
	serializable := eg.ToSerialization()

	for i, serErr := range serializable.Errors {
		fmt.Printf("Error %d:\n", i+1)
		fmt.Printf("  Type: %s\n", serErr.Type)
		fmt.Printf("  Message: %s\n", serErr.Message)
		fmt.Printf("  Stack frames: %d\n", len(serErr.StackTrace))
		if serErr.Metadata != nil {
			metadataJson, _ := json.MarshalIndent(serErr.Metadata, "  ", "  ")
			fmt.Printf("  Metadata: %s\n", string(metadataJson))
		}
		fmt.Println()
	}
}

func createErrorInFunction() *ewrap.Error {
	return ewrap.New("error created in function").WithMetadata("context", "example")
}
