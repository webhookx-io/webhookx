package dao

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

func newDB() *sqlx.DB {
	db := sqlx.MustOpen("sqlite3", ":memory:")
	schema := `
    CREATE TABLE test (
        id varchar(27) PRIMARY KEY,
        name TEXT,
        unknown TEXT,
        ws_id varchar(27)
    );`
	db.MustExec(schema)
	return db.Unsafe()
}

func initDB(db *sqlx.DB, entities []*TestEntity) {
	db.MustExec("DELETE FROM test")
	for _, entity := range entities {
		db.MustExec("INSERT INTO test (id, name, ws_id) VALUES (?, ?, ?)",
			entity.ID,
			entity.Name,
			entity.WorkspaceId)
	}
}

var _ = Describe("dao", Ordered, func() {
	db := newDB()
	dao := NewDAO[TestEntity](db, Options{Table: "test"})

	Context("Get", func() {
		BeforeAll(func() {
			initDB(db, []*TestEntity{
				{
					ID:   "00000000-0000-0000-0000-000000000000",
					Name: new("test"),
				},
			})
		})

		Context("errors", func() {
			It("returns no err when table has unknown columns", func() {
				entity, err := dao.Get(context.TODO(), "00000000-0000-0000-0000-000000000000")
				assert.NoError(GinkgoT(), err)
				assert.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), entity)
				assert.Equal(GinkgoT(), "test", *entity.Name)
				assert.Equal(GinkgoT(), "00000000-0000-0000-0000-000000000000", entity.ID)
			})
		})
	})

	Context("Delete", func() {
		It("return true, nil", func() {
			dao.db.MustExec(
				"INSERT INTO test (id, name, ws_id) VALUES (?, ?, ?)",
				"to-be-deleted",
				"test",
				"")
			ok, err := dao.Delete(context.TODO(), "to-be-deleted")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, ok)
		})

		It("returns false, nil", func() {
			ok, err := dao.Delete(context.TODO(), "")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), false, ok)
		})
	})

	Context("Iterator", func() {
		BeforeAll(func() {
			entities := []*TestEntity{
				{ID: "1", Name: nil},
				{ID: "2", Name: nil},
				{ID: "3", Name: nil},
				{ID: "4", Name: nil},
				{ID: "5", Name: nil},
			}
			initDB(db, entities)
		})

		It("iterate all entities", func() {
			var list []*TestEntity
			var q Query
			q.Limit = 2
			it := dao.Iterate(context.TODO(), &q)
			for it.Next() {
				v := it.Current()
				list = append(list, v)
			}
			assert.Equal(GinkgoT(), 5, len(list))
			assert.Equal(GinkgoT(), "5", list[0].ID)
			assert.Equal(GinkgoT(), "4", list[1].ID)
			assert.Equal(GinkgoT(), "3", list[2].ID)
			assert.Equal(GinkgoT(), "2", list[3].ID)
			assert.Equal(GinkgoT(), "1", list[4].ID)
		})
	})

})

func TestDAO(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "DAO")
}

func TestList(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.List(context.TODO(), nil)
	})
}

func TestCursor(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.Cursor(context.TODO(), nil)
	})

	t.Run("should panic when query.limit is negative", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query.limit must be positive" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		query := Query{
			Limit: -1,
		}
		dao.Cursor(context.TODO(), &query)
	})
}

func TestCount(t *testing.T) {
	dao := NewDAO[TestEntity](nil, Options{
		Table: "test_table",
	})

	t.Run("should panic when query is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != "query is nil" {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		dao.Count(context.TODO(), nil)
	})

}
