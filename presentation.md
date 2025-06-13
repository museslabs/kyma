---
title: Introduction
---

# Scout's honor: Making Laravel Search at Blazingly Fast Speeds with Typesense
Fanis Tharropoulos - Software Engineer @ Typesense


----

---
title: Typesense
preset: dark
notes: |
 Fanis Tahrropoulos
 * Giannis
---
# Typesense
- Open Source, typo-tolerant search engine (github.com/typesense/typesense)
&nbsp;
- Started in 2015
&nbsp;
- Built in C++
&nbsp;
- Designed for speed and ease of setup
&nbsp;
- 10B+ searches / month on Typesense Cloud (our SaaS product)
&nbsp;
- 10M+ downloads on Dockerhub
&nbsp;
- Clients for JavaScript, PHP, Python, Ruby, Go, Rust, C#, Java, Dart, Elixir, etc.
&nbsp;
- Official integrations with Laravel through Laravel Scout, Ruby on Rails, Firestore
----
---
title: Fanis
preset: dark
---
# Fanis Tharropoulos
- Software Engineer @ Typesense
&nbsp;
- Open Source Contributor (github.com/tharropoulos)
&nbsp;
- Building Kyma (github.com/museslabs/kyma) - a TUI presentation tool (you're looking at it right now!)
&nbsp;
 - Maintaining typesense-ts (github.com/tharropoulos/typesense-ts) - a TypeScript-native Typesense client with full type safety in mind
&nbsp;
----

---
title: Laravel Scout
---
# Laravel Scout
Laravel Scout is Laravel's official full-text search package that seamlessly integrates powerful search engines with your Eloquent models, making it easy to add fast, relevant search functionality to your Laravel applications.

**Key Features:**

• **Simple Integration** - Driver-based solution that syncs Eloquent models with search indexes automatically

• **Multiple Drivers** - Supports Typesense, Meilisearch, Algolia, MySQL/PostgreSQL (database driver), and local collection driver

• **Easy Setup** - Add the `Searchable` trait to models, configure driver, and start searching

• **Queue Support** - Async indexing for better performance with configurable queue connections

• **Flexible Searching** - Simple `Model::search('query')->get()` syntax with pagination, filters, and custom indexes

• **Auto-Sync** - Model observers automatically keep search indexes updated when records change

• **Customizable** - Override searchable data, index names per model

• **Soft Delete Support** - Handle soft-deleted records in search results with `withTrashed()` and `onlyTrashed()`

----
---
title: Why a Search Engine?
---

# Why use a Search Engine in the first place?
**Why not Postgres full-text search?**

Postgres full-text search is great for starting out and small datasets, but:

* It can be slow for large datasets, especially with complex queries
&nbsp;
* Introducing things like **typo-tolerance**, **synonyms**, **custom ranking**, **stopwords**, **curation rules**, **faceting**, **grouping**, **search presets** can get tricky
&nbsp;
* Adding **vector search** for semantic search requires additional setup and complexity
&nbsp;
* As things start to scale, cracks are going to show

A search engine like Typesense is built specifically for search, so it handles all of these things out of the box with a simple API.

----
---
title: Why Typesense?
---
# Why Typesense specifically?

- **Open Source** - Fully open source, so you can self-host or use our managed cloud service
&nbsp;
- **Speed** - Your indexes are saved in RAM, enabling fast search speeds, even with large datasets 
&nbsp;
- **Contributor Friendly** - Designed with contributors in mind, so you can easily add features or fix bugs
&nbsp;
- **Standardized API** - Consistent API across all languages, so you can switch clients easily
&nbsp;
- **AI Ready** - Supports vector search for semantic search, natural language queries, and conversational search for Chat-like experiences
&nbsp;
- **Well Documented** - Extensive documentation with examples and demos in multiple languages, making it easy to get started
----
---
title: Starting off with a Laravel Model
---
# Integrating Typesense with Laravel Scout
## The `Game` Model [available in Kaggle](https://www.kaggle.com/datasets/terencicp/steam-games-december-2023)
| Name | Type | Description      |
|------|------|------------------|
| name | string| The game's title          |
| release_date | datetime| The release date|
| price | float | Price in US dollars |
| positive | int | Number of positive reviews |
| negative | int | Number of negative reviews|
| app_id | int | Steam's id for the game|
| min_owners | int | Minimum number of possible owners for the game|
| max_owners | int | Maximum number of possible owners for the game|
| hltb_single | int | Hours needed to complete the game in Single Player |

----
```php --numbered
class Game extends Model
{
    use HasFactory;

    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = "games";

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = [
        "release_date" => "datetime",
        "price" => "float",
        "positive" => "integer",
        "negative" => "integer",
        "app_id" => "integer",
        "min_owners" => "integer",
        "max_owners" => "integer",
        "hltb_single" => "integer",
    ];
}
```

----
```php{1} --numbered
use Laravel\Scout\Searchable;

class Game extends Model
{
    use HasFactory, Searchable;

    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = "games";

    /**
     * Get the indexable data array for the model.
     *
     * @return array<string, mixed>
     */
    public function toSearchableArray()
    {
        return array_merge($this->toArray(), [
            "id" => (string) $this->id,
            "app_id" => (string) $this->app_id,
            "release_date" => $this->release_date?->getTimestamp()
        ]);
    }
  ...
}
```

----
```php{5} --numbered
use Laravel\Scout\Searchable;

class Game extends Model
{
    use HasFactory, Searchable;

    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = "games";

    /**
     * Get the indexable data array for the model.
     *
     * @return array<string, mixed>
     */
    public function toSearchableArray()
    {
        return array_merge($this->toArray(), [
            "id" => (string) $this->id,
            "app_id" => (string) $this->app_id,
            "release_date" => $this->release_date?->getTimestamp()
        ]);
    }
  ...
}
```
----
```php{19-26} --numbered
use Laravel\Scout\Searchable;

class Game extends Model
{
    use HasFactory, Searchable;

    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = "games";

    /**
     * Get the indexable data array for the model.
     *
     * @return array<string, mixed>
     */
    public function toSearchableArray()
    {
        return array_merge($this->toArray(), [
            "id" => (string) $this->id,
            "app_id" => (string) $this->app_id,
            "release_date" => $this->release_date?->getTimestamp()
        ]);
    }
  ...
}
```
