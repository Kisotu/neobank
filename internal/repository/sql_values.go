package repository

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func toPgUUID(id uuid.UUID) pgtype.UUID {
	var bytes [16]byte
	copy(bytes[:], id[:])
	return pgtype.UUID{Bytes: bytes, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	parsed, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func fromPgTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func fromNullablePgTime(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	v := ts.Time
	return &v
}

func toNullableText(value string) pgtype.Text {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func toNullableUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return toPgUUID(*id)
}
