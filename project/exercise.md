# Full-Text Search

The LIKE filter from the previous exercise works for substrings.
But try searching for "pizzas" when the menu item says "Margherita Pizza", and you'll get no results.

## LIKE Limitations

`LIKE '%pizzas%'` does a raw substring match. It has no concept of language:

- "pizzas" won't match "pizza", "running" won't match "run".
- All matches are equal, with no way to show the best matches first.
- The search term must appear verbatim somewhere in the text.

You might think full-text search requires a specialized database like Elasticsearch.
For some use cases it does, but PostgreSQL's built-in full-text search (FTS) can take you surprisingly far.
It's easier to operate (no extra infrastructure) and good enough for most applications.

We recommend starting with PostgreSQL for everything and looking for more specialized tools when you hit its limits.

## FTS Building Blocks

PostgreSQL full-text search works with a few key primitives.
Let's go through an example using a menu item called "Margherita Pizza".

```sql
to_tsvector('english', 'Margherita Pizza') @@ plainto_tsquery('english', 'pizzas')
-- returns true: "pizzas" stems to "pizza", which matches
```

1. **`to_tsvector('english', text)`** converts text into a searchable document.
It splits the text into words, removes stopwords like "the" and "a", and reduces words to their stems.
`to_tsvector('english', 'Margherita Pizza')` produces something like `'margherita':1 'pizza':2`.

2. **`plainto_tsquery('english', text)`** converts a search string into a query.
`plainto_tsquery('english', 'pizzas')` produces `'pizza'`, stemmed to match the document.
We use `plainto_tsquery` instead of `to_tsquery` because it handles raw user input safely.

3. **`@@`** is the match operator. It checks if a document matches a query.

## Extending the Query

Adding full-text search to your existing `ListMenuItemsWithRestaurant` query uses the same `CASE WHEN` + `sqlc.narg()` pattern from the previous exercise.
There are two additions:

**1. WHERE clause.**
The WHERE clause follows the same pattern as the `restaurant_name_filter`.
When `search_term` is NULL, skip the filter. When searching, return only matching rows:

```sql
AND (sqlc.narg(search_term)::text IS NULL
     OR to_tsvector('english', mi.name) @@ plainto_tsquery('english', sqlc.narg(search_term)::text))
```

**2. Relevance ordering option.**

A new `CASE WHEN` in the ORDER BY sorts results by relevance.
Notice how the relevance ordering comes first, before the existing ordering options:

```sql
CASE WHEN sqlc.narg(order_by)::text = 'relevance'
     THEN ts_rank(
         to_tsvector('english', mi.name),
         plainto_tsquery('english', sqlc.narg(search_term)::text)
     )
END DESC,
```

**`ts_rank(tsvector, tsquery)`** calculates a relevance score.
Higher scores mean better matches.
We'll use this to sort results so the best matches appear first.

## Go Code

The Go side barely changes. Since sqlc generates the new `SearchTerm` parameter from the SQL, all you do is pass it through:

```go
SearchTerm: filter.Search,
```

## GIN Index

You need a [GIN (Generalized Inverted Index)](https://www.postgresql.org/docs/current/gin.html) so PostgreSQL can use an index for `@@` queries instead of scanning every row.

For PostgreSQL to use a GIN index with `@@`, **the index expression must exactly match the expression in the WHERE clause.**
Our FTS query searches `mi.name`, so the index expression must be the same.

Your migration creates the index:

```sql
-- GIN index for full-text search on menu items
-- Indexes the tsvector for efficient @@ queries
CREATE INDEX idx_menu_items_fts
    ON orders.restaurant_menu_items
    USING gin (to_tsvector('english', name));
```

{{tip}}

If you want to learn more about the broader architectural pattern behind [read models](https://academy.threedots.tech/knowledge/read-model), check out [How to use basic CQRS in Go](https://threedots.tech/post/basic-cqrs-in-go/).

{{endtip}}

## Exercise

Exercise path: ./project

**Add full-text search to the read model** so users can search menu items by name, with results ranked by relevance.

1. **Create the migration** in `backend/orders/adapters/db/migrations/0004_fts_index.up.sql`. Create a GIN index on `to_tsvector('english', name)` for the `orders.restaurant_menu_items` table.
2. **Modify the SQL query** in `backend/orders/adapters/db/queries/read_models.sql`:
   - Add an FTS WHERE clause that activates only when `search_term` is provided.
   - Add a `relevance` ordering option in ORDER BY.
3. Run `task gen` to regenerate the sqlc and [OpenAPI](https://academy.threedots.tech/knowledge/openapi) code.
4. **Add `Search *string`** to `ListMenuItemsFilter` in `backend/orders/api/http/handler.go`.
5. Pass `request.Params.Search` to the filter in the `ListMenuItems` handler method.
6. **Pass `filter.Search` as `SearchTerm`** in `backend/orders/adapters/db/read_model.go` when calling `queries.ListMenuItems`.

Check `backend/orders/api/http/openapi.yaml` for the `search` parameter and the `relevance` enum value for `order_by` (already provided).

{{hints}}

{{hint 1}}

Here's one way to implement the full SQL query:

```sql
-- name: ListMenuItems :many
-- Lists menu items with optional restaurant name filter, optional full-text search, and dynamic ordering.
-- Uses CASE WHEN to support multiple ordering options in a single query.
SELECT
    mi.restaurant_menu_item_uuid AS menu_item_uuid,
    mi.name AS menu_item_name,
    mi.gross_price,
    r.currency,
    r.restaurant_uuid,
    r.name AS restaurant_name,
    CASE WHEN sqlc.narg(search_term)::text IS NOT NULL
         THEN ts_rank(
             to_tsvector('english', mi.name),
             plainto_tsquery('english', sqlc.narg(search_term)::text)
         )
         ELSE NULL
    END AS relevance
FROM orders.restaurant_menu_items mi
JOIN orders.restaurants r ON mi.restaurant_uuid = r.restaurant_uuid
WHERE mi.is_archived = false
  AND (sqlc.narg(search_term)::text IS NULL
       OR to_tsvector('english', mi.name) @@ plainto_tsquery('english', sqlc.narg(search_term)::text))
  AND (sqlc.narg(restaurant_name_filter)::text IS NULL
       OR LOWER(r.name) LIKE LOWER('%' || sqlc.narg(restaurant_name_filter)::text || '%'))
ORDER BY
    CASE WHEN sqlc.narg(order_by)::text = 'relevance'
         THEN ts_rank(
             to_tsvector('english', mi.name),
             plainto_tsquery('english', sqlc.narg(search_term)::text)
         )
    END DESC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default')
         THEN r.name END ASC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default')
         THEN mi.ordering END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_asc' THEN mi.gross_price END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_desc' THEN mi.gross_price END DESC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_asc' THEN mi.name END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_desc' THEN mi.name END DESC;
```

{{endhint}}

{{hint 2}}

The Go changes for the handler and read model can look like this:

{{codeFile "backend/orders/api/http/handler.go"}}

```go
type ListMenuItemsFilter struct {
	RestaurantName *string
	Search         *string
	OrderBy        *string
}
```

Then pass the search parameter in the `ListMenuItems` handler method:

```go
filter := ListMenuItemsFilter{
	RestaurantName: request.Params.RestaurantName,
	Search:         request.Params.Search,
	OrderBy:        orderBy,
}
```

In the read model, add `SearchTerm` to the params:

{{codeFile "backend/orders/adapters/db/read_model.go"}}

```go
rows, err := queries.ListMenuItems(ctx, dbmodels.ListMenuItemsParams{
	SearchTerm:           filter.Search,
	RestaurantNameFilter: filter.RestaurantName,
	OrderBy:              filter.OrderBy,
})
```

{{endhint}}

{{endhints}}
