---
id: getting-started-apply
title: Applying Schemas
slug: /cli/getting-started/applying-schemas
---

### Declarative migrations

In the previous section, we learned how to inspect an existing database and write
its schema as an Atlas DDL HCL file. In this section, we will learn how to use
the Atlas CLI to modify a database's schema. To do this, we will use Atlas's 
`atlas schema apply` command which takes a _declarative_ approach, that is,
we define the _desired_ end schema, and Atlas figures out a safe-way to alter
the database to get there. 

Let's start by viewing the help text for the `apply` command:

```shell
atlas schema apply --help
```
The documentation is printed out:
```text
Apply an atlas schema to a data source

Usage:
  atlas schema apply [flags]

Examples:

atlas schema apply -u "mysql://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply -u "mariadb://user:pass@tcp(localhost:3306)/dbname" -f atlas.hcl
atlas schema apply --url "postgres://user:pass@host:port/dbname?sslmode=disable" -f atlas.hcl
atlas schema apply -u "sqlite://file:ex1.db?_fk=1" -f atlas.hcl

Flags:
  -u, --url string    [driver://username:password@protocol(address)/dbname?param=value] Select data source using the url format
  -f, --file string   [/path/to/file] file containing schema
  -h, --help          help for apply
```
As you can see, similar to the `inspect` command, the `-d` flag is used to define the
URL to connect to the database, and an additional flag `-f` specifies the path to
the file containing the desired schema. 

### Adding new tables to our database

Let's modify our simplified blogging platform schema from the previous step by adding
a third table, `categories`. Each table will have an `id` and a `name`. In addition,
we will create an association table `post_categories` which creates a many-to-many
relationship between blog posts and categories:

![Blog ERD](https://atlasgo.io/uploads/images/blog-erd-2.png)

First, let's store the existing schema in a file named `atlas.hcl`:
```shell
atlas schema inspect -d "mysql://root:pass@tcp(localhost:3306)/example" > atlas.hcl
```
Next, add the following table definition to the file:
```hcl
table "categories" {
  schema = schema.example
  column "id" {
    null = false
    type = int
  }
  column "name" {
    null = true
    type = varchar(100)
  }
  primary_key {
    columns = [column.id]
  }
}
```

To add this table to our database, let's use the `atlas schema apply` command:
```shell
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl 
```
Atlas plans a migration (schema change) for us and prompts us to approve it:
```text
-- Planned Changes:
-- Create "categories" table
CREATE TABLE `example`.`categories` (`id` int NOT NULL, `name` varchar(100) NULL, PRIMARY KEY (`id`))
Use the arrow keys to navigate: ↓ ↑ → ←
? Are you sure?:
  ▸ Apply
    Abort
```
To apply the migration, press `ENTER`, and voila!
```text
✔ Apply
```

To verify that our new table was created, in our open `mysql` command-line from the 
previous step run:
```text
mysql> show create table categories;
+------------+------------------------------------------------------+
| Table      | Create Table                                                                                                                                                                 |
+------------+------------------------------------------------------+
| categories | CREATE TABLE `categories` (
  `id` int NOT NULL,
  `name` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci |
+------------+------------------------------------------------------
1 row in set (0.01 sec)
```

Amazing! Our new table was created. Next, let's define our association table,
add the following block to our `atlas.hcl` file:
```hcl
table "post_categories" {
    schema = schema.example
    column "post_id" {
        type = int
    }
    column "category_id" {
        type = int
    }
    foreign_key "post_category_post" {
        columns     = [column.post_id]
        ref_columns = [table.blog_posts.column.id]
    }
    foreign_key "post_category_category" {
        columns     = [column.category_id]
        ref_columns = [table.categories.column.id]
    }
}
```
This block defines the `post_categories` table with two columns `post_id` and `category_id`. 
In addition, two foreign-keys are created referencing the respective columns on the `blog_posts`
and `categories` tables. 

Let's try to apply the schema again, this time with the updated schema:
```text
atlas schema apply -d "mysql://root:pass@tcp(localhost:3306)/example" -f atlas.hcl
-- Planned Changes:
-- Create "post_categories" table
CREATE TABLE `example`.`post_categories` (`post_id` int NOT NULL, `category_id` int NOT NULL, CONSTRAINT `post_category_post` FOREIGN KEY (`post_id`) REFERENCES `example`.`blog_posts` (`id`), CONSTRAINT `post_category_category` FOREIGN KEY (`category_id`) REFERENCES `example`.`categories` (`id`))
✔ Apply
```

### Conclusion 

In this section, we've seen how to use the `atlas schema apply` command to migrate
the schema of an existing database to our desired state.