# jupyteach-cli

## `jupyteach pull` Command Workflow

The `jupyteach pull` command is designed to synchronize the local course content with the latest t version available on the server. It ensures that the local Git repository is aligned with the server's state, avoiding conflicts and ensuring consistency.

### Pull Workflow Steps

1. **Ensure Clean Workspace:**
   - Verify that there are no uncommitted changes or untracked files in the local Git workspace. The workspace must be completely clean to proceed with the pull operation.

2. **Initial Check with Server:**
   - Make a GET request to `/api/v1/push` to retrieve two pieces of data:
     - `remote_changes`: A boolean indicating if there have been changes on the server since the last synchronization.
     - `last_commit_SHA`: The SHA of the last commit known to the server.
   - If `remote_changes` is False and the `last_commit_SHA` exists in the local Git history, the local environment is up-to-date, and no further action is needed.
   - If the `last_commit_SHA` is not in the local Git history, prompt the user to perform a `git pull` to update their local repository.

3. **Executing Pull:**
   - If `remote_changes` is True, make a GET request to `/api/v1/pull` to download the current course content as a zip archive, which includes all the necessary files and the metadata.

4. **Extract and Update Local Files:**
   - Extract the zip archive received from the server.
   - Overwrite the local files with the contents of the archive, ensuring the local environment mirrors the server's content.

5. **Clean Up:**
   - Delete any local files that are not present in the extracted archive to ensure the local copy is an exact match of the server's content.

### Pull Implementation Notes

- The engineer should implement robust error handling to manage potential issues with network requests, file extraction, and file system operations.
- Provide clear and concise user feedback throughout the process, especially when instructing the user to perform additional Git operations or when changes are applied to the local environment.
- Include logging for critical steps to aid in troubleshooting and auditing the synchronization process.

## `jupyteach push` Command Workflow

The `jupyteach push` command is designed to synchronize local course content changes with the server, ensuring that only the latest and relevant changes are pushed, thereby maintaining the course content's integrity and consistency.

### Push Workflow Steps

1. **Ensure Clean Workspace:**
   - Confirm that there are no uncommitted changes or untracked files in the local Git workspace to prevent any loss of work or conflicts.

2. **Initial Check with Server:**
   - Perform a GET request to `/api/v1/push` to retrieve:
     - `remote_changes`: A boolean that indicates if there have been any changes on the server since the last sync.
     - `last_commit_SHA`: The commit SHA from the last successful push.
   - If `remote_changes` is True, halt the push operation and instruct the user to perform a `jupyteach pull` to synchronize the local and remote content.

3. **Determine Changes:**
   - If proceeding with the push, use `git diff <last_commit_SHA> HEAD --name-only` to identify which files have changed since the last sync.
   - Record the changed files in a `changes.json` manifest, categorizing them as added, modified, or deleted.

4. **Package Changes:**
   - Create a zip archive containing only the files listed in the `changes.json` manifest, ensuring that only relevant changes are sent to the server.

5. **POST Request with Changes:**
   - Send the `changes.json` manifest and the zip archive in a POST request to `/api/v1/push`.
   - Handle the server's response to ensure the push operation's success and provide feedback on the outcome.

### Push Implementation Notes

- Implement thorough error handling to address potential issues during the network request, file packaging, and server communication.
- Ensure that user feedback is informative, especially in scenarios where the push is halted or requires user intervention.
- Maintain detailed logging for each step to facilitate debugging and provide a clear operational history for auditing purposes.
