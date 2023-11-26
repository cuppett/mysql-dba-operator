/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package orm

import (
	"gorm.io/gorm"
	"time"
)

const DatabaseName = "zz_dba_operator"

type ManagedDatabase struct {
	Uuid         string `gorm:"primaryKey;size:36"`
	Namespace    string `gorm:"size:64"`
	Name         string `gorm:"size:64"`
	DatabaseName string `gorm:"size:64"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ManagedDatabase) TableName() string {
	return DatabaseName + ".managed_databases"
}

type ManagedUser struct {
	Uuid      string `gorm:"primaryKey;size:36"`
	Namespace string `gorm:"size:64"`
	Name      string `gorm:"size:64"`
	Username  string `gorm:"size:32"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ManagedUser) TableName() string {
	return DatabaseName + ".managed_users"
}

type DatabaseSchema struct {
	SchemaName          string `gorm:"size:64;column:SCHEMA_NAME"`
	DefaultCharacterSet string `gorm:"size:64;column:DEFAULT_CHARACTER_SET_NAME"`
	DefaultCollation    string `gorm:"size:64;column:DEFAULT_COLLATION_NAME"`
}

func (DatabaseSchema) TableName() string {
	return "INFORMATION_SCHEMA.SCHEMATA"
}

type MySqlUser struct {
	Host   string `gorm:"primaryKey;size:255;column:Host"`
	User   string `gorm:"primaryKey;size:32;column:User"`
	Plugin string `gorm:"size:64;column:plugin"`
}

func (MySqlUser) TableName() string {
	return "mysql.user"
}

type MySqlDb struct {
	Host string `gorm:"primaryKey;size:255;column:Host"`
	User string `gorm:"primaryKey;size:32;column:User"`
	Db   string `gorm:"size:64;column:Db"`
}

func (MySqlDb) TableName() string {
	return "mysql.db"
}

func DatabaseExists(gormDB *gorm.DB, name string) *DatabaseSchema {
	var schema DatabaseSchema
	gormDB.First(&schema, "SCHEMA_NAME = ?", name)

	if schema.SchemaName != "" {
		return &schema
	}
	return nil
}

func UserExists(gormDB *gorm.DB, name string) *MySqlUser {
	var user MySqlUser
	gormDB.First(&MySqlUser{}, "user = ?", name).Scan(&user)

	if user.User != "" {
		return &user
	}
	return nil
}
