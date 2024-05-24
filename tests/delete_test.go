package tests_test

import (
	"errors"
	"log"
	"os"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	. "gorm.io/gorm/utils/tests"
)

func TestDelete(t *testing.T) {
	users := []User{*GetUser("delete", Config{}), *GetUser("delete", Config{}), *GetUser("delete", Config{})}

	if err := DB.Create(&users).Error; err != nil {
		t.Errorf("errors happened when create: %v", err)
	}

	for _, user := range users {
		if user.ID == 0 {
			t.Fatalf("user's primary key should has value after create, got : %v", user.ID)
		}
	}

	if res := DB.Delete(&users[1]); res.Error != nil || res.RowsAffected != 1 {
		t.Errorf("errors happened when delete: %v, affected: %v", res.Error, res.RowsAffected)
	}

	var result User
	if err := DB.Where("id = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	if err := DB.Delete(&users[0]).Error; err != nil {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Delete(&User{}).Error; err != gorm.ErrMissingWhereClause {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Where("id = ?", users[0].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}

func TestDeleteWithTable(t *testing.T) {
	type UserWithDelete struct {
		gorm.Model
		Name string
	}

	DB.Table("deleted_users").Migrator().DropTable(UserWithDelete{})
	DB.Table("deleted_users").AutoMigrate(UserWithDelete{})

	user := UserWithDelete{Name: "delete1"}
	DB.Table("deleted_users").Create(&user)

	var result UserWithDelete
	if err := DB.Table("deleted_users").First(&result).Error; err != nil {
		t.Errorf("failed to find deleted user, got error %v", err)
	}

	AssertEqual(t, result, user)

	if err := DB.Table("deleted_users").Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user, got error %v", err)
	}

	var result2 UserWithDelete
	if err := DB.Table("deleted_users").First(&result2, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}

	var result3 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result3, user.ID).Error; err != nil {
		t.Fatalf("failed to find record, got error %v", err)
	}

	if err := DB.Table("deleted_users").Unscoped().Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user with unscoped, got error %v", err)
	}

	var result4 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result4, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}
}

func TestInlineCondDelete(t *testing.T) {
	user1 := *GetUser("inline_delete_1", Config{})
	user2 := *GetUser("inline_delete_2", Config{})
	DB.Save(&user1).Save(&user2)

	if DB.Delete(&User{}, user1.ID).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if err := DB.Where("name = ?", user1.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}

	if err := DB.Delete(&User{}, "name = ?", user2.Name).Error; err != nil {
		t.Errorf("No error should happen when delete a record, err=%s", err)
	} else if err := DB.Where("name = ?", user2.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}
}

func TestBlockGlobalDelete(t *testing.T) {
	if err := DB.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Errorf("should returns missing WHERE clause while deleting error")
	}

	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&User{}).Error; err != nil {
		t.Errorf("should returns no error while enable global update, but got err %v", err)
	}
}

func TestDeleteWithAssociations(t *testing.T) {
	user := GetUser("delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 1, "Pets": 2, "Toys": 4, "Company": 1, "Manager": 1, "Team": 1, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteAssociationsWithUnscoped(t *testing.T) {
	user := GetUser("unscoped_delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Unscoped().Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteSliceWithAssociations(t *testing.T) {
	users := []User{
		*GetUser("delete_slice_with_associations1", Config{Account: true, Pets: 4, Toys: 1, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 4}),
		*GetUser("delete_slice_with_associations2", Config{Account: true, Pets: 3, Toys: 2, Company: true, Manager: true, Team: 2, Languages: 2, Friends: 3}),
		*GetUser("delete_slice_with_associations3", Config{Account: true, Pets: 2, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 2}),
		*GetUser("delete_slice_with_associations4", Config{Account: true, Pets: 1, Toys: 4, Company: true, Manager: true, Team: 4, Languages: 4, Friends: 1}),
	}

	if err := DB.Create(users).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations).Delete(&users).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 4, "Pets": 10, "Toys": 10, "Company": 4, "Manager": 4, "Team": 10, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 4, "Manager": 4, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

// only sqlite, postgres, sqlserver support returning
func TestSoftDeleteReturning(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" && DB.Dialector.Name() != "postgres" && DB.Dialector.Name() != "sqlserver" {
		return
	}

	users := []*User{
		GetUser("delete-returning-1", Config{}),
		GetUser("delete-returning-2", Config{}),
		GetUser("delete-returning-3", Config{}),
	}
	DB.Create(&users)

	var results []User
	DB.Where("name IN ?", []string{users[0].Name, users[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&User{}).Where("name IN ?", []string{users[0].Name, users[1].Name, users[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestDeleteReturning(t *testing.T) {
	if DB.Dialector.Name() != "sqlite" && DB.Dialector.Name() != "postgres" && DB.Dialector.Name() != "sqlserver" {
		return
	}

	companies := []Company{
		{Name: "delete-returning-1"},
		{Name: "delete-returning-2"},
		{Name: "delete-returning-3"},
	}
	DB.Create(&companies)

	var results []Company
	DB.Where("name IN ?", []string{companies[0].Name, companies[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&Company{}).Where("name IN ?", []string{companies[0].Name, companies[1].Name, companies[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestNestedDelete(t *testing.T) {
	DB.Logger = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             1,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	})
	// user has 2 pets, 2 friends, 1 manager, 2 team
	// use's pets has toy
	// friend has 1 pets, 1 toys, 1 tools, 1 account
	// team has 2 pets, 2 toys, 2 tools, 1 account
	// manager has 3 pets, 3 toys, 3 tools, 1 account
	user := User{
		Name: "nest-delete-user",
		Age:  18,
		Account: Account{
			Number: "nest-delete-user-account",
		},
		Pets: []*Pet{
			{
				Name: "nest-delete-user-pet1",
				Toy: Toy{
					Name: "nest-delete-user-pet1-toy",
				},
			},
			{
				Name: "nest-delete-user-pet2",
				Toy: Toy{
					Name: "nest-delete-user-pet2-toy",
				},
			},
		},
		Manager: &User{
			Name: "nest-delete-user-manager",
			Account: Account{
				Number: "nest-delete-user-manager-account",
			},
			Pets: []*Pet{
				{
					Name: "nest-delete-user-manager-pet1",
					Toy: Toy{
						Name: "nest-delete-user-manager-pet1-toy",
					},
				},
				{
					Name: "nest-delete-user-manager-pet2",
					Toy: Toy{
						Name: "nest-delete-user-manager-pet2-toy",
					},
				},
				{
					Name: "nest-delete-user-manager-pet3",
					Toy: Toy{
						Name: "nest-delete-user-manager-pet3-toy",
					},
				},
			},
			Toys: []Toy{
				{
					Name: "nest-delete-user-manager-toy1",
				},
				{
					Name: "nest-delete-user-manager-toy2",
				},
				{
					Name: "nest-delete-user-manager-toy3",
				},
			},
			Tools: []Tools{
				{
					Name: "nest-delete-user-manager-tool1",
				},
				{
					Name: "nest-delete-user-manager-tool2",
				},
				{
					Name: "nest-delete-user-manager-tool3",
				},
			},
		},
		Team: []User{
			{
				Name: "nest-delete-user-team1",
				Account: Account{
					Number: "nest-delete-user-team1-account",
				},
				Pets: []*Pet{
					{
						Name: "nest-delete-user-team1-pet1",
					},
					{
						Name: "nest-delete-user-team1-pet2",
					},
				},
				Toys: []Toy{
					{
						Name: "nest-delete-user-team1-toy1",
					},
					{
						Name: "nest-delete-user-team1-toy2",
					},
				},
				Tools: []Tools{
					{
						Name: "nest-delete-user-team1-tool1",
					},
					{
						Name: "nest-delete-user-team1-tool2",
					},
				},
			},
			{
				Name: "nest-delete-user-team2",
				Account: Account{
					Number: "nest-delete-user-team2-account",
				},
				Pets: []*Pet{
					{
						Name: "nest-delete-user-team2-pet1",
					},
					{
						Name: "nest-delete-user-team2-pet2",
					},
				},
				Toys: []Toy{
					{
						Name: "nest-delete-user-team2-toy1",
					},
					{
						Name: "nest-delete-user-team2-toy2",
					},
				},
				Tools: []Tools{
					{
						Name: "nest-delete-user-team2-tool1",
					},
					{
						Name: "nest-delete-user-team2-tool2",
					},
				},
			},
		},
		Friends: []*User{
			{
				Name: "nest-delete-user-friend1",
				Account: Account{
					Number: "nest-delete-user-friend1-account",
				},
				Pets: []*Pet{
					{
						Name: "nest-delete-user-friend1-pet1",
					},
				},
				Toys: []Toy{
					{
						Name: "nest-delete-user-friend1-toy1",
					},
				},
				Tools: []Tools{
					{
						Name: "nest-delete-user-friend1-tool1",
					},
				},
			},
			{
				Name: "nest-delete-user-friend2",
				Account: Account{
					Number: "nest-delete-user-friend2-account",
				},
				Pets: []*Pet{
					{
						Name: "nest-delete-user-friend2-pet1",
					},
				},
				Toys: []Toy{
					{
						Name: "nest-delete-user-friend2-toy1",
					},
				},
				Tools: []Tools{
					{
						Name: "nest-delete-user-friend2-tool1",
					},
				},
			},
		},
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user1, got error %v", err)
	}

	if err := DB.Select("Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 1, "Pets": 2, "Toys": 4, "Company": 1, "Manager": 1, "Team": 1, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}
