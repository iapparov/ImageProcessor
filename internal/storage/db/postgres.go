package db

import (
	"context"
	"database/sql"
	"fmt"
	wbdb "github.com/wb-go/wbf/dbpg"
	wbretry "github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
)

type Postgres struct {
	db  *wbdb.DB
	cfg *config.RetrysConfig
}

func NewPostgres(cfg *config.AppConfig) (*Postgres, error) {
	masterDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBConfig.Master.Host,
		cfg.DBConfig.Master.Port,
		cfg.DBConfig.Master.User,
		cfg.DBConfig.Master.Password,
		cfg.DBConfig.Master.DBName,
	)

	slaveDSNs := make([]string, 0, len(cfg.DBConfig.Slaves))
	for _, slave := range cfg.DBConfig.Slaves {
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			slave.Host,
			slave.Port,
			slave.User,
			slave.Password,
			slave.DBName,
		)
		slaveDSNs = append(slaveDSNs, dsn)
	}
	var opts wbdb.Options
	opts.ConnMaxLifetime = cfg.DBConfig.ConnMaxLifetime
	opts.MaxIdleConns = cfg.DBConfig.MaxIdleConns
	opts.MaxOpenConns = cfg.DBConfig.MaxOpenConns
	db, err := wbdb.New(masterDSN, slaveDSNs, &opts)
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to connect to Postgres")
		return nil, err
	}
	wbzlog.Logger.Info().Msg("Connected to Postgres")
	return &Postgres{db: db, cfg: &cfg.RetrysConfig}, nil
}

func (p *Postgres) Close() error {
	err := p.db.Master.Close()
	if err != nil {
		wbzlog.Logger.Debug().Msg("Failed to close Postgres connection")
		return err
	}
	for _, slave := range p.db.Slaves {
		if slave != nil {
			err := slave.Close()
			if err != nil {
				wbzlog.Logger.Debug().Msg("Failed to close Postgres slave connection")
				return err
			}
		}
	}
	return nil
}

func (s *Postgres) SaveImage(img *domain.Image) error {
	ctx := context.Background()
	query := `
		INSERT INTO images (id, created_at, status, format, name, watermark, resize_height, resize_width)
		VALUES($1, $2, 'created', $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query,
		img.ID,
		img.CreatedAt,
		img.Format,
		img.Name,
		img.Watermark,
		img.Resize.Height,
		img.Resize.Width,
	)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute insert comment query")
		return err
	}
	return nil
}

func (s *Postgres) GetImage(id string) (*domain.Image, error) {
	ctx := context.Background()
	query := `
		SELECT id, created_at, status, format, name, watermark, resize_height, resize_width
		FROM images
		WHERE id = $1 AND status != 'deleted'
	`
	row, err := s.db.QueryRowWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query, id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute get image query")
		return nil, err
	}
	var img domain.Image
	var resizeHeight sql.NullInt64
	var resizeWidth sql.NullInt64
	err = row.Scan(
		&img.ID,
		&img.CreatedAt,
		&img.Status,
		&img.Format,
		&img.Name,
		&img.Watermark,
		&resizeHeight,
		&resizeWidth,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("image not found")
		}
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute get image query (scan)")
		return nil, err
	}
	if resizeHeight.Valid && resizeWidth.Valid {
		img.Resize = &domain.Resize{
			Width:  int(resizeWidth.Int64),
			Height: int(resizeHeight.Int64),
		}
	} else {
		img.Resize = &domain.Resize{
			Width:  0,
			Height: 0,
		}
	}
	return &img, nil
}

func (s *Postgres) DeleteImage(id string) error {
	ctx := context.Background()
	query := `
		UPDATE images
		SET status = 'deleted'
		WHERE id = $1
	`
	_, err := s.db.ExecWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query, id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute delete image query")
		return err
	}
	return nil

}

func (s *Postgres) SetProcessing(id string) error {
	ctx := context.Background()
	query := `
		UPDATE images
		SET status = 'processing'
		WHERE id = $1
	`
	_, err := s.db.ExecWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query, id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute set processing image query")
		return err
	}
	return nil
}

func (s *Postgres) SetProcessed(id string) error {
	ctx := context.Background()
	query := `
		UPDATE images
		SET status = 'processed'
		WHERE id = $1
	`
	_, err := s.db.ExecWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query, id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute set processed image query")
		return err
	}
	return nil

}

func (s *Postgres) UploadInProducer() ([]domain.Image, error) {
	ctx := context.Background()
	query := `
		SELECT id, created_at, status, format, name, watermark, resize_height, resize_width
		FROM images
		WHERE status = 'created'
	`
	rows, err := s.db.QueryWithRetry(ctx, wbretry.Strategy{Attempts: s.cfg.Attempts, Delay: s.cfg.Delay, Backoff: s.cfg.Backoffs}, query)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to execute upload in producer query")
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to close rows")
		}
	}()
	var images []domain.Image
	for rows.Next() {
		var img domain.Image
		var resizeHeight sql.NullInt64
		var resizeWidth sql.NullInt64
		err := rows.Scan(
			&img.ID,
			&img.CreatedAt,
			&img.Status,
			&img.Format,
			&img.Name,
			&img.Watermark,
			&resizeHeight,
			&resizeWidth,
		)
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to scan image row")
			return nil, err
		}
		if resizeHeight.Valid && resizeWidth.Valid {
			img.Resize = &domain.Resize{
				Width:  int(resizeWidth.Int64),
				Height: int(resizeHeight.Int64),
			}
		} else {
			img.Resize = &domain.Resize{
				Width:  0,
				Height: 0,
			}
		}
		images = append(images, img)
	}
	return images, nil
}
