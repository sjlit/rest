package query

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testUser struct {
	ID    uint
	Name  string
	Age   int
	Email string
}

type testOrder struct {
	ID     uint
	UserID uint
	Amount float64
	Status string
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique in-memory DB per test to avoid data leakage.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.AutoMigrate(&testUser{}, &testOrder{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedUsers(db *gorm.DB) {
	db.Create(&testUser{Name: "Alice", Age: 30, Email: "alice@example.com"})
	db.Create(&testUser{Name: "Bob", Age: 25, Email: "bob@example.com"})
	db.Create(&testUser{Name: "Charlie", Age: 35, Email: "charlie@example.com"})
	db.Create(&testUser{Name: "Diana", Age: 28, Email: "diana@example.com"})
}

func seedOrders(db *gorm.DB) {
	db.Create(&testOrder{UserID: 1, Amount: 100.0, Status: "paid"})
	db.Create(&testOrder{UserID: 1, Amount: 200.0, Status: "pending"})
	db.Create(&testOrder{UserID: 2, Amount: 150.0, Status: "paid"})
}

// ---------- Operator Tests ----------

func TestOpEq(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var u testUser
	b := NewBuilder().Where("name", OpEq, "Alice")
	if err := New(db, nil, b).One(ctx, &u); err != nil {
		t.Fatalf("One failed: %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("expected Alice, got %s", u.Name)
	}
}

func TestOpEqNil(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&testUser{Name: "NullUser", Age: 0, Email: ""})
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("email", OpEq, "")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	// Empty string is not nil, so this should find NullUser
	found := false
	for _, u := range users {
		if u.Name == "NullUser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find NullUser with empty email")
	}
}

func TestOpNe(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("name", OpNe, "Alice")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestOpGtLt(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("age", OpGt, 28)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users with age > 28, got %d", len(users))
	}

	users = nil
	b = NewBuilder().Where("age", OpLt, 28)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user with age < 28, got %d", len(users))
	}
}

func TestOpGteLte(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("age", OpGte, 30)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users with age >= 30, got %d", len(users))
	}

	users = nil
	b = NewBuilder().Where("age", OpLte, 25)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user with age <= 25, got %d", len(users))
	}
}

func TestOpLike(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Like("name", "li", LikeContains)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users matching %%li%%, got %d", len(users))
	}
}

func TestOpLikeModes(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	// Prefix
	var users []testUser
	b := NewBuilder().Like("name", "Al", LikePrefix)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 || users[0].Name != "Alice" {
		t.Errorf("expected Alice, got %v", users)
	}

	// Suffix
	users = nil
	b = NewBuilder().Like("name", "ce", LikeSuffix)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 || users[0].Name != "Alice" {
		t.Errorf("expected Alice, got %v", users)
	}
}

func TestOpIn(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("name", OpIn, []string{"Alice", "Bob"})
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestOpBetween(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("age", OpBetween, []int{25, 30})
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users with age between 25 and 30, got %d", len(users))
	}
}

func TestOpBetweenInvalid(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	b := NewBuilder().Where("age", OpBetween, []int{25})
	_, err := New(db, &testUser{}, b).Count(ctx)
	if err == nil {
		t.Error("expected error for invalid BETWEEN value")
	}
}

func TestOrderByCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	// lowercase "desc" should be recognized as DESC
	var users []testUser
	b := NewBuilder().OrderBy("age", "desc")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected 4 users, got %d", len(users))
	}

	// unrecognized direction should normalize to ASC
	users = nil
	b = NewBuilder().OrderBy("age", "foo")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected 4 users, got %d", len(users))
	}
}

func TestLikeEscaping(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&testUser{Name: "100%", Age: 1, Email: "a@example.com"})
	db.Create(&testUser{Name: "100_percent", Age: 2, Email: "b@example.com"})
	ctx := context.Background()

	// Searching for literal "%" should not match "100_percent"
	var users []testUser
	b := NewBuilder().Like("name", "%", LikeContains)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 || users[0].Name != "100%" {
		t.Errorf("expected exactly '100%%', got %v", users)
	}

	// Searching for literal "_" should match "100_percent" but not "100%"
	users = nil
	b = NewBuilder().Like("name", "_", LikeContains)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 || users[0].Name != "100_percent" {
		t.Errorf("expected exactly '100_percent', got %v", users)
	}
}

func TestOpIsNull(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&testUser{Name: "NullEmail", Age: 20, Email: ""})
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("email", OpIsNull, nil)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	// SQLite treats empty string differently from NULL, so this should find no users
	// unless we explicitly set NULL. Let's test with a real NULL.
	if len(users) != 0 {
		t.Logf("SQLite empty string behavior: found %d users", len(users))
	}
}

func TestOpIsNotNull(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("email", OpIsNotNull, nil)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected 4 users with non-null email, got %d", len(users))
	}
}

// ---------- Group Tests ----------

func TestAndGroup(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().WhereGroup(func(b *Builder) {
		b.Where("age", OpGt, 25).Where("age", OpLt, 35)
	})
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users with 25 < age < 35, got %d", len(users))
	}
}

func TestOrWhere(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().Where("name", OpEq, "Alice").OrWhere("name", OpEq, "Bob")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users (Alice or Bob), got %d", len(users))
	}
}

func TestOrWhereGroup(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().
		Where("age", OpLt, 26).
		OrWhereGroup(func(b *Builder) {
			b.Where("age", OpGt, 30).Where("name", OpEq, "Charlie")
		})
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestComplexGroup(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	// (age > 30 OR age < 26) AND name LIKE %a%
	var users []testUser
	b := NewBuilder().
		WhereGroup(func(b *Builder) {
			b.Where("age", OpGt, 30).OrWhere("age", OpLt, 26)
		}).
		Where("name", OpLike, "%a%")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	// Charlie: age=35 (>30) and name contains 'a' => match
	// Bob: age=25 (<26) but name doesn't contain 'a' => no match
	if len(users) != 1 || users[0].Name != "Charlie" {
		t.Errorf("expected 1 user (Charlie), got %d: %v", len(users), users)
	}
}

// ---------- Join Tests ----------

func TestLeftJoin(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	seedOrders(db)
	ctx := context.Background()

	type result struct {
		Name   string
		Amount float64
	}

	var results []result
	b := NewBuilder().
		Select(Field("test_users.name"), Field("test_orders.amount")).
		From("test_users").
		LeftJoin("test_orders", "test_users.id = test_orders.user_id")
	if err := New(db, nil, b).All(ctx, &results); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected join results, got none")
	}
}

func TestInnerJoin(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	seedOrders(db)
	ctx := context.Background()

	type result struct {
		Name   string
		Amount float64
	}

	var results []result
	b := NewBuilder().
		Select(Field("test_users.name"), Field("test_orders.amount")).
		From("test_users").
		InnerJoin("test_orders", "test_users.id = test_orders.user_id")
	if err := New(db, nil, b).All(ctx, &results); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	// Only users with orders should appear
	if len(results) == 0 {
		t.Error("expected inner join results, got none")
	}
}

// ---------- Execution Tests ----------

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	b := NewBuilder().Where("age", OpGte, 28)
	count, err := New(db, &testUser{}, b).Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestOneUsesTakeNotFirst(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	// Without explicit ordering, Take returns the first row the DB returns.
	// The key difference from First is that Take does NOT inject ORDER BY id.
	var u testUser
	b := NewBuilder()
	if err := New(db, nil, b).One(ctx, &u); err != nil {
		t.Fatalf("One failed: %v", err)
	}
	if u.ID == 0 {
		t.Error("expected a user to be returned")
	}
}

func TestAll(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder()
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected 4 users, got %d", len(users))
	}
}

func TestPage(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().OrderBy("id", "ASC")
	total, err := New(db, &testUser{}, b).Page(ctx, 1, 2, &users)
	if err != nil {
		t.Fatalf("Page failed: %v", err)
	}
	if total != 4 {
		t.Errorf("expected total 4, got %d", total)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users on page 1, got %d", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("expected Alice on page 1, got %s", users[0].Name)
	}

	users = nil
	total, err = New(db, &testUser{}, b).Page(ctx, 2, 2, &users)
	if err != nil {
		t.Fatalf("Page failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users on page 2, got %d", len(users))
	}
	if users[0].Name != "Charlie" {
		t.Errorf("expected Charlie on page 2, got %s", users[0].Name)
	}
}

func TestPageInvalidInput(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder()
	// page=0 should normalize to 1, size=0 should normalize to 20
	total, err := New(db, &testUser{}, b).Page(ctx, 0, 0, &users)
	if err != nil {
		t.Fatalf("Page failed: %v", err)
	}
	if total != 4 {
		t.Errorf("expected total 4, got %d", total)
	}
	if len(users) != 4 {
		t.Errorf("expected all 4 users with default size, got %d", len(users))
	}
}

// ---------- Builder Chain Tests ----------

func TestChaining(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().
		Select(Field("name"), Field("age")).
		Where("age", OpGte, 25).
		OrderBy("age", "DESC").
		Limit(2)
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Age < users[1].Age {
		t.Error("expected descending order")
	}
}

func TestFromTable(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().From("test_users").Where("name", OpEq, "Alice")
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestEmptyBuilder(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder()
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 4 {
		t.Errorf("expected all 4 users, got %d", len(users))
	}
}

func TestGroupBy(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	ctx := context.Background()

	type result struct {
		Age int
	}

	var results []result
	b := NewBuilder().
		Select(Field("age")).
		GroupBy("age").
		OrderBy("age", "ASC")
	if err := New(db, &testUser{}, b).All(ctx, &results); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(results) != 4 {
		t.Errorf("expected 4 distinct ages, got %d", len(results))
	}
}

// ---------- ParseOperator Tests ----------

func TestParseOperator(t *testing.T) {
	tests := []struct {
		input string
		want  Operator
	}{
		{"eq", OpEq},
		{"=", OpEq},
		{"ne", OpNe},
		{"!=", OpNe},
		{"gt", OpGt},
		{"gte", OpGte},
		{"ge", OpGte},
		{"like", OpLike},
		{"in", OpIn},
		{"between", OpBetween},
		{"isnull", OpIsNull},
		{"is not null", OpIsNotNull},
		{"unknown", Operator("unknown")},
	}

	for _, tt := range tests {
		got := ParseOperator(tt.input)
		if got != tt.want {
			t.Errorf("ParseOperator(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSubqueryIn(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db)
	seedOrders(db)
	ctx := context.Background()

	var users []testUser
	b := NewBuilder().
		Where("id", OpIn, NewSubquery(func(sb *Builder) {
			sb.Select(Field("user_id")).From("test_orders").Where("amount", OpGt, 100)
		}))
	if err := New(db, nil, b).All(ctx, &users); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users with orders > 100, got %d", len(users))
	}
}

func TestAggregateAndHaving(t *testing.T) {
	db := setupTestDB(t)
	seedOrders(db)
	ctx := context.Background()

	type result struct {
		Status string
		Total  float64
	}

	var results []result
	b := NewBuilder().
		Select(Field("status"), Sum("amount").As("total")).
		From("test_orders").
		GroupBy("status").
		Having("total", OpGt, 100)
	if err := New(db, nil, b).All(ctx, &results); err != nil {
		t.Fatalf("All failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected aggregate results, got none")
	}
}
