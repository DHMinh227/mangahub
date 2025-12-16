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
