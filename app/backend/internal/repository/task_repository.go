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

type TaskCount struct {
	Total int
	Done  int
}

type TaskRepository struct {
	col *mongo.Collection
}

func NewTaskRepository(db *mongo.Database) *TaskRepository {
	return &TaskRepository{col: db.Collection("tasks")}
}

func (r *TaskRepository) Create(ctx context.Context, t *models.Task) error {
	_, err := r.col.InsertOne(ctx, t)
	return err
}

func (r *TaskRepository) CreateMany(ctx context.Context, tasks []models.Task) error {
	docs := make([]any, 0, len(tasks))
	for _, t := range tasks {
		docs = append(docs, t)
	}
	_, err := r.col.InsertMany(ctx, docs)
	return err
}

func (r *TaskRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Task, error) {
	var t models.Task
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepository) FindByProject(ctx context.Context, projectID primitive.ObjectID) ([]models.Task, error) {
	return r.find(ctx, bson.M{"projectId": projectID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
}

// FindOpenAssignedTo returns unfinished tasks assigned to the user, earliest due date first.
func (r *TaskRepository) FindOpenAssignedTo(ctx context.Context, userID primitive.ObjectID, limit int64) ([]models.Task, error) {
	return r.find(ctx,
		bson.M{"assigneeId": userID, "status": bson.M{"$ne": "done"}},
		options.Find().SetSort(bson.D{{Key: "dueDate", Value: 1}}).SetLimit(limit))
}

func (r *TaskRepository) Replace(ctx context.Context, t *models.Task) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": t.ID}, t)
	return err
}

func (r *TaskRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *TaskRepository) DeleteByProject(ctx context.Context, projectID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"projectId": projectID})
	return err
}

func (r *TaskRepository) Unassign(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.col.UpdateMany(ctx, bson.M{"assigneeId": userID}, bson.M{"$unset": bson.M{"assigneeId": ""}})
	return err
}

// CountsByProject returns total and done task counts per project.
func (r *TaskRepository) CountsByProject(ctx context.Context, projectIDs []primitive.ObjectID) (map[primitive.ObjectID]TaskCount, error) {
	out := map[primitive.ObjectID]TaskCount{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	cur, err := r.col.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"projectId": bson.M{"$in": projectIDs}}},
		bson.M{"$group": bson.M{
			"_id":   "$projectId",
			"total": bson.M{"$sum": 1},
			"done":  bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "done"}}, 1, 0}}},
		}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Total int                `bson:"total"`
		Done  int                `bson:"done"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = TaskCount{Total: row.Total, Done: row.Done}
	}
	return out, nil
}

// CountByStatus returns the number of tasks per status across the given projects.
func (r *TaskRepository) CountByStatus(ctx context.Context, projectIDs []primitive.ObjectID) (map[string]int, error) {
	out := map[string]int{}
	cur, err := r.col.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"projectId": bson.M{"$in": projectIDs}}},
		bson.M{"$group": bson.M{"_id": "$status", "n": bson.M{"$sum": 1}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Status string `bson:"_id"`
		N      int    `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Status] = row.N
	}
	return out, nil
}

func (r *TaskRepository) CountOverdue(ctx context.Context, projectIDs []primitive.ObjectID, now time.Time) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{
		"projectId": bson.M{"$in": projectIDs},
		"status":    bson.M{"$ne": "done"},
		"dueDate":   bson.M{"$lt": now},
	})
}

func (r *TaskRepository) find(ctx context.Context, filter bson.M, opts ...*options.FindOptions) ([]models.Task, error) {
	cur, err := r.col.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	tasks := []models.Task{}
	if err := cur.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}
