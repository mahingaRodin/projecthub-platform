package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"qt-devops-go-mongo/backend/internal/models"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	_, err := r.col.InsertOne(ctx, u)
	if mongo.IsDuplicateKeyError(err) {
		return ErrDuplicate
	}
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

func (r *UserRepository) FindAll(ctx context.Context) ([]models.User, error) {
	return r.find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
}

func (r *UserRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	return r.find(ctx, bson.M{"_id": bson.M{"$in": ids}})
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{})
}

func (r *UserRepository) CountByIDs(ctx context.Context, ids []primitive.ObjectID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	return r.col.CountDocuments(ctx, bson.M{"_id": bson.M{"$in": ids}})
}

func (r *UserRepository) UpdateRole(ctx context.Context, id primitive.ObjectID, role models.Role) error {
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"role": role}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) find(ctx context.Context, filter any, opts ...*options.FindOptions) ([]models.User, error) {
	cur, err := r.col.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	users := []models.User{}
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}
