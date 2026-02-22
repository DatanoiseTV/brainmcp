package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// runInteractiveCLI starts an interactive command-line interface for testing the memory system.
// Users can manually test all available operations without needing an MCP client.
func (a *App) runInteractiveCLI(ctx context.Context) {
	fmt.Println(WelcomeMsg)
	fmt.Println(HelpMsg)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n" + PrompStr)
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToLower(parts[0])
		switch cmd {
		case "exit":
			a.gracefulShutdown()
			return

		case "list":
			a.cliList(ctx)

		case "wipe":
			a.cliWipe(ctx)

		case "ask":
			if len(parts) < 2 {
				fmt.Println("Usage: ask <question>")
				continue
			}
			a.cliAsk(ctx, strings.Join(parts[1:], " "))

		case "remember":
			if len(parts) < 3 {
				fmt.Println("Usage: remember <id> <content>")
				continue
			}
			a.cliRemember(ctx, parts[1], strings.Join(parts[2:], " "))

		case "search":
			if len(parts) < 2 {
				fmt.Println("Usage: search <query>")
				continue
			}
			a.cliSearch(ctx, strings.Join(parts[1:], " "))

		case "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: delete <id>")
				continue
			}
			a.cliDelete(ctx, parts[1])

		case "tag":
			if len(parts) < 3 {
				fmt.Println("Usage: tag <memory_id> <tag>")
				continue
			}
			a.cliAddTag(ctx, parts[1], parts[2])

		case "tags":
			a.cliListTags(ctx)

		case "context":
			if len(parts) < 2 {
				fmt.Println("Usage: context <list|create|switch>")
				continue
			}
			subCmd := strings.ToLower(parts[1])
			switch subCmd {
			case "list":
				a.cliListContexts(ctx)
			case "create":
				if len(parts) < 4 {
					fmt.Println("Usage: context create <id> <name>")
					continue
				}
				a.cliCreateContext(ctx, parts[2], parts[3])
			case "switch":
				if len(parts) < 3 {
					fmt.Println("Usage: context switch <id>")
					continue
				}
				a.cliSwitchContext(ctx, parts[2])
			default:
				fmt.Println("Unknown context command. Try: context list|create|switch")
			}

		case "save":
			a.cliSaveToDisk(ctx)

		default:
			fmt.Println(UnknownCmdMsg)
		}
	}
}

// cliRemember executes the remember operation from CLI.
func (a *App) cliRemember(ctx context.Context, id, content string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": id, "content": content}
	res, _ := a.rememberHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliSearch executes the search operation from CLI.
func (a *App) cliSearch(ctx context.Context, query string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": query}
	res, _ := a.searchHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliAsk executes the ask_brain operation from CLI.
func (a *App) cliAsk(ctx context.Context, question string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"question": question}
	res, err := a.askBrainHandler(ctx, req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if res.IsError {
		fmt.Printf("Error: %v\n", res.Content[0].(mcp.TextContent).Text)
	} else {
		fmt.Println(res.Content[0].(mcp.TextContent).Text)
	}
}

// cliDelete executes the delete operation from CLI.
func (a *App) cliDelete(ctx context.Context, id string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": id}
	res, _ := a.deleteHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliList executes the list operation from CLI.
func (a *App) cliList(ctx context.Context) {
	req := mcp.CallToolRequest{}
	res, _ := a.listHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliWipe executes the wipe operation from CLI.
func (a *App) cliWipe(ctx context.Context) {
	req := mcp.CallToolRequest{}
	res, _ := a.wipeHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliAddTag adds a tag to a memory from CLI.
func (a *App) cliAddTag(ctx context.Context, memoryID, tag string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"memory_id": memoryID, "tag": tag}
	res, _ := a.addTagHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliListTags lists all tags from CLI.
func (a *App) cliListTags(ctx context.Context) {
	req := mcp.CallToolRequest{}
	res, _ := a.listTagsHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliListContexts lists all contexts from CLI.
func (a *App) cliListContexts(ctx context.Context) {
	req := mcp.CallToolRequest{}
	res, _ := a.listContextsHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliCreateContext creates a new context from CLI.
func (a *App) cliCreateContext(ctx context.Context, id, name string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": id, "name": name, "description": ""}
	res, _ := a.createContextHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliSwitchContext switches to a different context from CLI.
func (a *App) cliSwitchContext(ctx context.Context, contextID string) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"context_id": contextID, "client_id": ""}
	res, _ := a.switchContextHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}

// cliSaveToDisk saves the database and context state from CLI.
func (a *App) cliSaveToDisk(ctx context.Context) {
	req := mcp.CallToolRequest{}
	res, _ := a.saveToDiskHandler(ctx, req)
	fmt.Println(res.Content[0].(mcp.TextContent).Text)
}
