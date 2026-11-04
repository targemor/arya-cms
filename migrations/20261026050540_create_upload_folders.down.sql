DROP INDEX IF EXISTS upload_files_folder_idx;
ALTER TABLE upload_files DROP COLUMN IF EXISTS folder_id;
DROP TABLE IF EXISTS upload_folders;
