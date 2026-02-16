package services

import (
	"context"
	"time"

	"github.com/mh-airlines/afs-engine/internal/config"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database manages MongoDB connection and operations
type Database struct {
	client    *mongo.Client
	afsDB     *mongo.Database
	refDB     *mongo.Database
	config    *config.Config
}

// NewDatabase creates a new database instance
func NewDatabase(cfg *config.Config) *Database {
	return &Database{
		config: cfg,
	}
}

// Connect establishes MongoDB connection
func (d *Database) Connect(ctx context.Context) error {
	log.Info("Connecting to MongoDB...")

	clientOptions := options.Client().
		ApplyURI(d.config.MongoDB.URI).
		SetMaxPoolSize(d.config.MongoDB.MaxPoolSize).
		SetMinPoolSize(d.config.MongoDB.MinPoolSize).
		SetMaxConnIdleTime(d.config.MongoDB.MaxConnIdleTime).
		SetServerSelectionTimeout(d.config.MongoDB.ServerSelectionTimeout)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	d.client = client
	

	d.afsDB = client.Database(d.config.MongoDB.Database)           // afs_db
	d.refDB = client.Database(d.config.MongoDB.ReferenceDatabase)  // master_reference

	log.WithFields(log.Fields{
		"afs_database": d.config.MongoDB.Database,
		"ref_database": d.config.MongoDB.ReferenceDatabase,
	}).Info("MongoDB connected successfully")

	if err := d.setupIndexes(ctx); err != nil {
		log.WithError(err).Warn("Failed to setup some indexes")
	}

	return nil
}

// setupIndexes creates required indexes
func (d *Database) setupIndexes(ctx context.Context) error {
	log.Info("Setting up database indexes...")

	mfsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "startDate", Value: 1},
				{Key: "endDate", Value: 1},
				{Key: "scheduleStatus", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "flightNo", Value: 1},
				{Key: "seasonId", Value: 1},
				{Key: "itineraryVarId", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "scheduleStatus", Value: 1}},
		},
	}

	_, err := d.afsDB.Collection("master_flights").Indexes().CreateMany(ctx, mfsIndexes)
	if err != nil {
		log.WithError(err).Warn("Failed to create MFS indexes")
	}

	afsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "flightDate", Value: 1},
				{Key: "flightNo", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "deliveryStatus", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // TTL index
		},
	}

	_, err = d.afsDB.Collection("active_flights").Indexes().CreateMany(ctx, afsIndexes)
	if err != nil {
		log.WithError(err).Warn("Failed to create AFS indexes")
	}

	// ===== MASTER_REFERENCE Indexes =====
	
	// Airlines Index (in master_reference)
	airlineIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "isActive", Value: 1}},
		},
	}

	_, err = d.refDB.Collection("airlines").Indexes().CreateMany(ctx, airlineIndexes)
	if err != nil {
		log.WithError(err).Warn("Failed to create airline indexes")
	}

	// Airports Index (in master_reference)
	airportIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "iataCode", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "countryCode", Value: 1}},
		},
	}

	_, err = d.refDB.Collection("iata_airports").Indexes().CreateMany(ctx, airportIndexes)
	if err != nil {
		log.WithError(err).Warn("Failed to create airport indexes")
	}

	log.Info("Database indexes setup completed")
	return nil
}

// ===== AFS Database Methods =====

// GetAFSDB returns the AFS database instance
func (d *Database) GetAFSDB() *mongo.Database {
	return d.afsDB
}

// GetAFSCollection returns a collection from afs_db
func (d *Database) GetAFSCollection(name string) *mongo.Collection {
	return d.afsDB.Collection(name)
}

// ===== Reference Database Methods =====

// GetRefDB returns the Reference database instance
func (d *Database) GetRefDB() *mongo.Database {
	return d.refDB
}

// GetRefCollection returns a collection from master_reference
func (d *Database) GetRefCollection(name string) *mongo.Collection {
	return d.refDB.Collection(name)
}

func (d *Database) GetDB() *mongo.Database {
	return d.afsDB
}

func (d *Database) GetCollection(name string) *mongo.Collection {
	return d.afsDB.Collection(name)
}

// Close closes the database connection
func (d *Database) Close(ctx context.Context) error {
	if d.client != nil {
		log.Info("Closing MongoDB connection...")
		return d.client.Disconnect(ctx)
	}
	return nil
}

// HealthCheck performs database health check
func (d *Database) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.client.Ping(ctx, nil)
}