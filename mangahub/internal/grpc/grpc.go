package grpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	pb "mangahub/proto/manga"
)

type GRPCMangaServer struct {
	pb.UnimplementedMangaServiceServer
	DB *sql.DB
}

func (s *GRPCMangaServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {

	row := s.DB.QueryRow(`
        SELECT id, title, author, genres, status, total_chapters, description
        FROM manga
        WHERE id = ?
    `, req.Id)

	var m pb.MangaResponse
	var genresText string

	if err := row.Scan(&m.Id, &m.Title, &m.Author, &genresText, &m.Status, &m.TotalChapters, &m.Description); err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(genresText), &m.Genres)

	return &m, nil
}

func (s *GRPCMangaServer) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {

	rows, err := s.DB.Query(`
        SELECT id, title, author, status
        FROM manga
        WHERE (title LIKE '%' || ? || '%' OR ? = '')
        AND (genres LIKE '%' || ? || '%' OR ? = '')
        AND (status = ? OR ? = '')
        LIMIT ?
    `, req.Query, req.Query,
		req.Genre, req.Genre,
		req.Status, req.Status,
		req.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := &pb.SearchResponse{}
	for rows.Next() {
		var r pb.SearchResult
		rows.Scan(&r.Id, &r.Title, &r.Author, &r.Status)
		resp.Results = append(resp.Results, &r)
	}

	return resp, nil
}

func (s *GRPCMangaServer) UpdateProgress(ctx context.Context, req *pb.ProgressRequest) (*pb.ProgressResponse, error) {

	// 1) Validate manga exists + get total chapters
	var total int32
	err := s.DB.QueryRow(`
        SELECT total_chapters FROM manga WHERE id = ?
    `, req.MangaId).Scan(&total)

	if err == sql.ErrNoRows {
		return &pb.ProgressResponse{Message: "Manga does not exist"}, nil
	}
	if err != nil {
		return nil, err
	}

	// 2) Validate chapter range
	if req.CurrentChapter < 1 {
		return &pb.ProgressResponse{Message: "Chapter must be at least 1"}, nil
	}

	if req.CurrentChapter > total {
		return &pb.ProgressResponse{
			Message: fmt.Sprintf("Invalid chapter. Max: %d", total),
		}, nil
	}

	// 3) If valid → save progress
	_, err = s.DB.Exec(`
        INSERT INTO user_progress (user_id, manga_id, current_chapter)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id, manga_id)
        DO UPDATE SET current_chapter = excluded.current_chapter
    `, req.UserId, req.MangaId, req.CurrentChapter)

	if err != nil {
		return &pb.ProgressResponse{Message: "Failed to update"}, err
	}

	return &pb.ProgressResponse{Message: "Progress updated!"}, nil
}

func (s *GRPCMangaServer) GetProgress(ctx context.Context, req *pb.GetProgressRequest) (*pb.GetProgressResponse, error) {

	row := s.DB.QueryRow(`
        SELECT current_chapter 
        FROM user_progress 
        WHERE user_id = ? AND manga_id = ?
    `, req.UserId, req.MangaId)

	var ch int32
	err := row.Scan(&ch)
	if err == sql.ErrNoRows {
		return &pb.GetProgressResponse{
			Exists:         false,
			CurrentChapter: 0,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &pb.GetProgressResponse{
		Exists:         true,
		CurrentChapter: ch,
	}, nil
}
