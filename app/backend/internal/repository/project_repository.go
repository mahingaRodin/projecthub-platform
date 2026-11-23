package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"qt-devops-go-mongo/backend/internal/models"
)

type ProjectRepository struct {
	col *mongo.Collection
}

func NewProjectRepository(db *mongo.Database) *ProjectRepository {
	return &ProjectRepository{col: db.Collection("projects")}
}

func (r *ProjectRepository) Create(ctx context.Context, p *models.Project) error {
	_, err := r.col.InsertOne(ctx, p)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicate
	}
	return err
}

func (r *ProjectRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Project, error) {
	var p models.Project
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindAll returns every project, most recently updated first.
func (r *ProjectRepository) FindAll(ctx context.Context) ([]models.Project, error) {
	return r.find(ctx, bson.M{})
}

// FindByParticipant returns projects the user owns or is a member of.
func (r *ProjectRepository) FindByParticipant(ctx context.Context, userID primitive.ObjectID) ([]models.Project, error) {
	return r.find(ctx, bson.M{"$or": bson.A{bson.M{"ownerId": userID}, bson.M{"memberIds": userID}}})
}

func (r *ProjectRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{})
}

func (r *ProjectRepository) Replace(ctx context.Context, p *models.Project) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": p.ID}, p)
	return err
}

func (r *ProjectRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// Touch bumps updatedAt, e.g. after one of the project's tasks changed.
func (r *ProjectRepository) Touch(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"updatedAt": time.Now().UTC()}})
	return err
}

func (r *ProjectRepository) TransferOwnership(ctx context.Context, from, to primitive.ObjectID) error {
	_, err := r.col.UpdateMany(ctx, bson.M{"ownerId": from}, bson.M{"$set": bson.M{"ownerId": to}})
	return err
}

func (r *ProjectRepository) RemoveMember(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.col.UpdateMany(ctx, bson.M{"memberIds": userID}, bson.M{"$pull": bson.M{"memberIds": userID}})
	return err
}

func (r *ProjectRepository) find(ctx context.Context, filter bson.M) ([]models.Project, error) {
	cur, err := r.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	projects := []models.Project{}
	if err := cur.All(ctx, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}
