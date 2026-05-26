package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const chatHistoryLimit = 100

type ChatMessage struct {
	Role           string   `json:"role"`
	Content        string   `json:"content"`
	ContextRelPath string   `json:"contextRelPath"`
	SourcePaths    []string `json:"sourcePaths"`
	CreatedAt      string   `json:"createdAt"`
}

type ChatHistoryStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultChatHistoryStore() *ChatHistoryStore {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}

	return NewChatHistoryStore(filepath.Join(configDir, "NexusAugenticStudio", "chat-history.json"))
}

func NewChatHistoryStore(path string) *ChatHistoryStore {
	return &ChatHistoryStore{path: path}
}

func (s *ChatHistoryStore) List(workspaceRoot string) ([]ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.read()
	if err != nil {
		return nil, err
	}

	return cloneChatMessages(items[workspaceHistoryKey(workspaceRoot)]), nil
}

func (s *ChatHistoryStore) AppendPair(workspaceRoot string, userMessage ChatMessage, assistantMessage ChatMessage) ([]ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.read()
	if err != nil {
		return nil, err
	}

	key := workspaceHistoryKey(workspaceRoot)
	now := time.Now().UTC()
	user := normalizeChatMessage(userMessage, "user", now.Format(time.RFC3339Nano))
	assistant := normalizeChatMessage(assistantMessage, "assistant", now.Add(time.Millisecond).Format(time.RFC3339Nano))
	if assistant.CreatedAt == user.CreatedAt {
		assistant.CreatedAt = nextChatTimestamp(user.CreatedAt)
	}
	next := append(cloneChatMessages(items[key]), user, assistant)
	if len(next) > chatHistoryLimit {
		next = next[len(next)-chatHistoryLimit:]
	}

	items[key] = next
	if err := s.write(items); err != nil {
		return nil, err
	}

	return cloneChatMessages(next), nil
}

func nextChatTimestamp(createdAt string) string {
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return time.Now().UTC().Add(time.Millisecond).Format(time.RFC3339Nano)
	}
	return parsed.Add(time.Millisecond).Format(time.RFC3339Nano)
}

func (s *ChatHistoryStore) Clear(workspaceRoot string) ([]ChatMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.read()
	if err != nil {
		return nil, err
	}

	delete(items, workspaceHistoryKey(workspaceRoot))
	if err := s.write(items); err != nil {
		return nil, err
	}

	return []ChatMessage{}, nil
}

func (s *ChatHistoryStore) Search(workspaceRoot string, query string) ([]ChatMessage, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []ChatMessage{}, nil
	}

	messages, err := s.List(workspaceRoot)
	if err != nil {
		return nil, err
	}

	results := []ChatMessage{}
	for _, message := range messages {
		haystack := strings.ToLower(message.Content + "\n" + message.ContextRelPath + "\n" + strings.Join(message.SourcePaths, "\n"))
		if strings.Contains(haystack, query) {
			results = append(results, message)
		}
	}
	return results, nil
}

func (s *ChatHistoryStore) read() (map[string][]ChatMessage, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string][]ChatMessage{}, nil
	}
	if err != nil {
		return nil, err
	}

	items := map[string][]ChatMessage{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *ChatHistoryStore) write(items map[string][]ChatMessage) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func normalizeChatMessage(message ChatMessage, fallbackRole string, fallbackCreatedAt string) ChatMessage {
	message.Role = strings.TrimSpace(message.Role)
	if message.Role == "" {
		message.Role = fallbackRole
	}
	message.Content = strings.TrimSpace(message.Content)
	message.ContextRelPath = strings.TrimSpace(message.ContextRelPath)
	message.SourcePaths = cleanChatSourcePaths(message.SourcePaths)
	message.CreatedAt = strings.TrimSpace(message.CreatedAt)
	if message.CreatedAt == "" {
		message.CreatedAt = fallbackCreatedAt
	}
	return message
}

func cleanChatSourcePaths(paths []string) []string {
	cleaned := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		cleaned = append(cleaned, path)
	}
	return cleaned
}

func cloneChatMessages(messages []ChatMessage) []ChatMessage {
	if len(messages) == 0 {
		return []ChatMessage{}
	}

	next := make([]ChatMessage, len(messages))
	copy(next, messages)
	return next
}

func workspaceHistoryKey(workspaceRoot string) string {
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		absRoot = workspaceRoot
	}

	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(absRoot))))
	return hex.EncodeToString(sum[:])
}
