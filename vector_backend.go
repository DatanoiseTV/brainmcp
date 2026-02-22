package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/philippgille/chromem-go"
	"github.com/qdrant/go-client/qdrant"
)

// VectorBackend defines the interface for vector storage implementations.
type VectorBackend interface {
	// AddDocument stores a single document with its embedding.
	AddDocument(ctx context.Context, document chromem.Document) error

	// AddDocuments stores multiple documents with embeddings.
	AddDocuments(ctx context.Context, documents []chromem.Document, concurrency int) error

	// Query searches for similar documents.
	Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error)

	// QueryEmbedding searches using a pre-computed embedding.
	QueryEmbedding(ctx context.Context, queryEmbedding []float32, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error)

	// GetByID retrieves a document by ID.
	GetByID(ctx context.Context, id string) (chromem.Document, error)

	// Delete removes documents by IDs.
	Delete(ctx context.Context, where, whereDocument map[string]string, ids ...string) error

	// ClearAll removes all documents from the store.
	ClearAll(ctx context.Context) error

	// Count returns the number of documents.
	Count() int

	// Close closes the backend connection.
	Close() error

	// SaveToDisk persists the vector store to disk.
	SaveToDisk() error
}

// LocalVectorStore wraps chromem-go as our local backend.
type LocalVectorStore struct {
	collection *chromem.Collection
	db         *chromem.DB
	embFunc    chromem.EmbeddingFunc
	logger     *log.Logger
	mu         sync.RWMutex
}

// NewLocalVectorStore creates a new local vector store using chromem-go.
func NewLocalVectorStore(dbPath string, embFunc chromem.EmbeddingFunc, logger *log.Logger) (*LocalVectorStore, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	// Load or create persistent database
	db, err := chromem.NewPersistentDB(dbPath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem database: %w", err)
	}

	// Create or get collection
	collection, err := db.GetOrCreateCollection("memories", nil, embFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	lvs := &LocalVectorStore{
		collection: collection,
		db:         db,
		embFunc:    embFunc,
		logger:     logger,
	}

	logger.Printf("Initialized local vector store with chromem-go (file: %s)", dbPath)
	return lvs, nil
}

// AddDocuments adds documents to the collection.
func (lvs *LocalVectorStore) AddDocuments(ctx context.Context, documents []chromem.Document, concurrency int) error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	return lvs.collection.AddDocuments(ctx, documents, concurrency)
}

// AddDocument adds a single document to the collection.
func (lvs *LocalVectorStore) AddDocument(ctx context.Context, document chromem.Document) error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	return lvs.collection.AddDocument(ctx, document)
}

// Query performs semantic search.
func (lvs *LocalVectorStore) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error) {
	lvs.mu.RLock()
	defer lvs.mu.RUnlock()

	return lvs.collection.Query(ctx, queryText, nResults, where, whereDocument)
}

// QueryEmbedding searches using a pre-computed embedding vector.
func (lvs *LocalVectorStore) QueryEmbedding(ctx context.Context, queryEmbedding []float32, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error) {
	lvs.mu.RLock()
	defer lvs.mu.RUnlock()

	return lvs.collection.QueryEmbedding(ctx, queryEmbedding, nResults, where, whereDocument)
}

// GetByID retrieves a document by ID.
func (lvs *LocalVectorStore) GetByID(ctx context.Context, id string) (chromem.Document, error) {
	lvs.mu.RLock()
	defer lvs.mu.RUnlock()

	return lvs.collection.GetByID(ctx, id)
}

// Delete removes documents by IDs.
func (lvs *LocalVectorStore) Delete(ctx context.Context, where, whereDocument map[string]string, ids ...string) error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	return lvs.collection.Delete(ctx, where, whereDocument, ids...)
}

// ClearAll removes all documents from the collection.
func (lvs *LocalVectorStore) ClearAll(ctx context.Context) error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	// Delete and recreate the collection
	collectionName := lvs.collection.Name
	if err := lvs.db.DeleteCollection(collectionName); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	col, err := lvs.db.GetOrCreateCollection(collectionName, nil, lvs.embFunc)
	if err != nil {
		return fmt.Errorf("failed to recreate collection: %w", err)
	}

	lvs.collection = col
	lvs.logger.Printf("Cleared all documents from collection %q", collectionName)
	return nil
}

// Count returns the total number of documents.
func (lvs *LocalVectorStore) Count() int {
	lvs.mu.RLock()
	defer lvs.mu.RUnlock()

	return lvs.collection.Count()
}

// Close exports the database to disk.
func (lvs *LocalVectorStore) Close() error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	if lvs.db != nil {
		if err := lvs.db.ExportToFile("", true, ""); err != nil {
			return fmt.Errorf("failed to export database before closing: %w", err)
		}
		return nil
	}
	return nil
}

// SaveToDisk persists the local vector store to disk.
func (lvs *LocalVectorStore) SaveToDisk() error {
	lvs.mu.Lock()
	defer lvs.mu.Unlock()

	if lvs.db != nil {
		if err := lvs.db.ExportToFile("", true, ""); err != nil {
			return fmt.Errorf("failed to export database to disk: %w", err)
		}
	}
	return nil
}

// QdrantVectorStore implements VectorBackend using Qdrant remote service.
type QdrantVectorStore struct {
	client    *qdrant.Client
	collName  string
	embFunc   chromem.EmbeddingFunc
	logger    *log.Logger
	mu        sync.RWMutex
	vectorDim uint64
}

// DocumentStore stores metadata for Qdrant points.
type DocumentStore struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

// NewQdrantVectorStore connects to a Qdrant instance and initializes a collection.
func NewQdrantVectorStore(host string, port int, apiKey string, useTLS bool, vectorDim int, embFunc chromem.EmbeddingFunc, logger *log.Logger) (*QdrantVectorStore, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	// Connect to Qdrant
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	qvs := &QdrantVectorStore{
		client:    client,
		collName:  "brainmcp-memories",
		embFunc:   embFunc,
		logger:    logger,
		vectorDim: uint64(vectorDim),
	}

	// FIX 1: ListCollections now returns []string directly, not a struct.
	collections, err := client.ListCollections(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list Qdrant collections: %w", err)
	}

	collectionExists := false
	for _, name := range collections {
		if name == qvs.collName {
			collectionExists = true
			break
		}
	}

	if !collectionExists {
		logger.Printf("Creating Qdrant collection: %s (vector_size: %d)", qvs.collName, qvs.vectorDim)
		err = client.CreateCollection(context.Background(), &qdrant.CreateCollection{
			CollectionName: qvs.collName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     qvs.vectorDim,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Qdrant collection: %w", err)
		}
	}

	logger.Printf("Connected to Qdrant at %s:%d (collection: %s)", host, port, qvs.collName)
	return qvs, nil
}

// AddDocuments adds documents to Qdrant.
func (qvs *QdrantVectorStore) AddDocuments(ctx context.Context, documents []chromem.Document, concurrency int) error {
	qvs.mu.Lock()
	defer qvs.mu.Unlock()

	if len(documents) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(documents))

	for i, doc := range documents {
		// Generate embedding
		embedding, err := qvs.embFunc(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to embed document %q: %w", doc.ID, err)
		}

		// FIX 2: Use qdrant.NewVectors(slice...) instead of struct literal with unknown field.
		vectors := qdrant.NewVectors(embedding...)

		// FIX 3: Serialize document metadata into the payload map properly.
		//        Remove the unused `payload` variable.
		docStore := DocumentStore{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
		}
		payloadBytes, err := json.Marshal(docStore)
		if err != nil {
			return fmt.Errorf("failed to marshal document %q: %w", doc.ID, err)
		}

		// FIX 4: Use qdrant.NewIDNum(uint64) helper instead of struct literal with unknown field.
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(hashStringToUint64(doc.ID)),
			Vectors: vectors,
			Payload: qdrant.NewValueMap(map[string]any{
				"payload": string(payloadBytes),
			}),
		}
	}

	_, err := qvs.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qvs.collName,
		Points:         points,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points to Qdrant: %w", err)
	}

	qvs.logger.Printf("Added %d documents to Qdrant", len(documents))
	return nil
}

// AddDocument adds a single document to Qdrant.
func (qvs *QdrantVectorStore) AddDocument(ctx context.Context, document chromem.Document) error {
	return qvs.AddDocuments(ctx, []chromem.Document{document}, 1)
}

// GetByID retrieves a document by ID.
func (qvs *QdrantVectorStore) GetByID(ctx context.Context, id string) (chromem.Document, error) {
	qvs.mu.RLock()
	defer qvs.mu.RUnlock()

	pointID := hashStringToUint64(id)

	// FIX 5: Use Ids field with qdrant.NewIDNum helpers instead of PointsSelector struct.
	points, err := qvs.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: qvs.collName,
		Ids:            []*qdrant.PointId{qdrant.NewIDNum(pointID)},
	})
	if err != nil {
		return chromem.Document{}, fmt.Errorf("failed to get point from Qdrant: %w", err)
	}

	if len(points) == 0 {
		return chromem.Document{}, fmt.Errorf("document %q not found", id)
	}

	// Extract document metadata from payload
	if payloadVal, ok := points[0].Payload["payload"]; ok {
		if stringVal, ok := payloadVal.Kind.(*qdrant.Value_StringValue); ok {
			var docStore DocumentStore
			if err := json.Unmarshal([]byte(stringVal.StringValue), &docStore); err == nil {
				return chromem.Document{
					ID:       docStore.ID,
					Content:  docStore.Content,
					Metadata: docStore.Metadata,
				}, nil
			}
		}
	}

	return chromem.Document{}, fmt.Errorf("failed to decode document %q", id)
}

// Query is not natively supported on QdrantVectorStore without a separate embed call;
// it embeds the query text first then delegates to QueryEmbedding.
func (qvs *QdrantVectorStore) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error) {
	embedding, err := qvs.embFunc(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	return qvs.QueryEmbedding(ctx, embedding, nResults, where, whereDocument)
}

// QueryEmbedding searches Qdrant using a pre-computed embedding vector.
func (qvs *QdrantVectorStore) QueryEmbedding(ctx context.Context, queryEmbedding []float32, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error) {
	qvs.mu.RLock()
	defer qvs.mu.RUnlock()

	limit := uint64(nResults)
	result, err := qvs.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: qvs.collName,
		Query:          qdrant.NewQueryDense(queryEmbedding),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}

	results := make([]chromem.Result, 0, len(result))
	for _, hit := range result {
		payloadVal, ok := hit.Payload["payload"]
		if !ok {
			continue
		}
		stringVal, ok := payloadVal.Kind.(*qdrant.Value_StringValue)
		if !ok {
			continue
		}
		var docStore DocumentStore
		if err := json.Unmarshal([]byte(stringVal.StringValue), &docStore); err != nil {
			continue
		}
		results = append(results, chromem.Result{
			ID:        docStore.ID,
			Metadata:  docStore.Metadata,
			Embedding: nil,
			Content:   docStore.Content,
			// SimilarityScore is not directly available from QueryPoints scored results
			// but can be approximated; leave as zero or populate if needed.
		})
	}

	return results, nil
}

// Delete removes documents from Qdrant.
// FIX 6: Use client.Delete() (not DeletePoints) with qdrant.NewPointsSelector helper.
func (qvs *QdrantVectorStore) Delete(ctx context.Context, where, whereDocument map[string]string, ids ...string) error {
	qvs.mu.Lock()
	defer qvs.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	// Build a slice of *qdrant.PointId for the selector
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = qdrant.NewIDNum(hashStringToUint64(id))
	}

	_, err := qvs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qvs.collName,
		Points:         qdrant.NewPointsSelector(pointIDs...),
	})

	if err != nil {
		return fmt.Errorf("failed to delete points from Qdrant: %w", err)
	}

	qvs.logger.Printf("Deleted %d documents from Qdrant", len(ids))
	return nil
}

// ClearAll removes all documents from Qdrant by deleting and recreating the collection.
func (qvs *QdrantVectorStore) ClearAll(ctx context.Context) error {
	qvs.mu.Lock()
	defer qvs.mu.Unlock()

	// Delete collection
	err := qvs.client.DeleteCollection(ctx, qvs.collName)
	if err != nil {
		return fmt.Errorf("failed to delete Qdrant collection: %w", err)
	}

	// Recreate collection
	err = qvs.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: qvs.collName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     qvs.vectorDim,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to recreate Qdrant collection: %w", err)
	}

	qvs.logger.Printf("Cleared all documents from Qdrant collection %q", qvs.collName)
	return nil
}

// Count returns the number of documents in Qdrant.
// FIX 7: Use client.GetCollectionInfo() (not CollectionInfo) and dereference *uint64 PointsCount.
func (qvs *QdrantVectorStore) Count() int {
	qvs.mu.RLock()
	defer qvs.mu.RUnlock()

	info, err := qvs.client.GetCollectionInfo(context.Background(), qvs.collName)
	if err != nil {
		qvs.logger.Printf("Warning: Failed to get collection info: %v", err)
		return 0
	}

	if info.PointsCount == nil {
		return 0
	}
	return int(*info.PointsCount)
}

// Close closes the Qdrant connection.
func (qvs *QdrantVectorStore) Close() error {
	return qvs.client.Close()
}

// SaveToDisk is a no-op for Qdrant since data is persisted server-side.
func (qvs *QdrantVectorStore) SaveToDisk() error {
	return nil
}

// NewVectorBackend factory function that returns the appropriate backend based on configuration.
func NewVectorBackend(cfg *Config, embFunc chromem.EmbeddingFunc, logger *log.Logger) (VectorBackend, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	// Check for Qdrant configuration
	if cfg != nil && cfg.Qdrant.Host != "" {
		qdrantHost := cfg.Qdrant.Host
		qdrantPort := cfg.Qdrant.Port
		if qdrantPort == 0 {
			qdrantPort = 6334 // Default Qdrant gRPC port
		}
		qdrantAPIKey := cfg.Qdrant.APIKey
		useTLS := cfg.Qdrant.UseTLS
		vectorDim := cfg.Qdrant.VectorDimension
		if vectorDim == 0 {
			vectorDim = 768
		}

		logger.Printf("Attempting to use Qdrant backend: %s:%d", qdrantHost, qdrantPort)
		return NewQdrantVectorStore(qdrantHost, qdrantPort, qdrantAPIKey, useTLS, vectorDim, embFunc, logger)
	}

	// Use local chromem-go backend as default
	dataDir := os.Getenv("BRAINMCP_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = home + "/.brainmcp"
	}

	return NewLocalVectorStore(dataDir+"/brain_memory.bin", embFunc, logger)
}

// hashStringToUint64 converts a string ID to uint64 for Qdrant point IDs.
func hashStringToUint64(s string) uint64 {
	hash := uint64(5381)
	for i := 0; i < len(s); i++ {
		hash = ((hash << 5) + hash) + uint64(s[i])
	}
	return hash
}