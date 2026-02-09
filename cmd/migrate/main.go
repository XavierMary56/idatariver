package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"idatariver-finapi/internal/util"
)

func main() {
	ctx := context.Background()
	dsn := util.Getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	migrationsDir := util.Getenv("MIGRATIONS_DIR", "./migrations")
	files, err := listSQL(migrationsDir)
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		ver := filepath.Base(f)
		applied, err := isApplied(ctx, pool, ver)
		if err != nil {
			panic(err)
		}
		if applied {
			fmt.Printf("skip %s\n", ver)
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			panic(err)
		}
		sql := string(b)
		fmt.Printf("apply %s\n", ver)
		if _, err := pool.Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("migration %s failed: %w", ver, err))
		}
		if _, err := pool.Exec(ctx, `insert into schema_migrations(version, applied_at) values ($1,$2)`, ver, time.Now().UTC()); err != nil {
			panic(err)
		}
	}

	fmt.Println("migrations ok")
}

func listSQL(dir string) ([]string, error) {
	out := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	// schema_migrations table might not exist yet; try and return false.
	var cnt int
	err := pool.QueryRow(ctx, `select count(1) from information_schema.tables where table_name='schema_migrations'`).Scan(&cnt)
	if err != nil {
		return false, err
	}
	if cnt == 0 {
		return false, nil
	}
	var n int
	if err := pool.QueryRow(ctx, `select count(1) from schema_migrations where version=$1`, version).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
