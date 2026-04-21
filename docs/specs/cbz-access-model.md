# Page Access

Because CBZ is a ZIP file, random access to individual pages is possible without loading the entire archive into memory.
ZIP's central directory (stored at the end of the file) maps each entry to its byte offset,
so a single page can be extracted with a seek + decompress of that entry alone.
