
# Book Identity

Book identity is managed by a UUID stored inside `folio.json` within the CBZ.
This decouples identity from file location, so moving a CBZ between subdirectories does not lose its metadata.

**Scan logic:**

```
Open CBZ
  ├─ folio.json exists → read UUID
  └─ folio.json missing → generate UUID, write folio.json into CBZ

UUID exists in DB → update source path only (file may have moved)
UUID missing in DB → insert new record
```