package grpc

import (
	"context"
	"database/sql"
	"encoding/json"

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

	_, err := s.DB.Exec(`
        INSERT INTO progress (user_id, manga_id, current_chapter)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id, manga_id)
        DO UPDATE SET current_chapter = excluded.current_chapter
    `, req.UserId, req.MangaId, req.CurrentChapter)

	if err != nil {
		return &pb.ProgressResponse{Message: "Failed to update"}, err
	}

	return &pb.ProgressResponse{Message: "Progress updated!"}, nil
}
