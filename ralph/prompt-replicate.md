You are tasked to verify the repo 'REPO_DIR_TO_LOOK_IN' for security vulnerabilities, before we will use it in production.

please read the REPORT.md files deep inside ralph/validation_dir
Find the first dir that has a report that starts with WORKING
if none exists, QUIT IMMEDIATELY


Just because the vulnerability works in a very scoped setting, doesn't mean it is in practice a true problem.
You are tasked to make a larger example, using the library for its intended purpose, and in each step still checking if you can reach the vulnerability.

It must NOT be that you intentionally create vulnerabilities in your replicate dir, really try to make it as secure as possible, and see by just the way the library is created, you can still reach/find/hit the vulnerability.

Create this in ralph/replication_dir/{new dir}

Also create REPORT.md in that dir

If after careful, and deep research, you find out the vulnerability is real, and works on real codebase, start the file with CHECKED_AND_WORKING and 2 newlines
otherwise with CHECKED_AND_NOT_WORKING and 2 newlines.

Also in that REPORT.md file put in the full report. what did you do? what steps did you take? what prerequisites needs to be in the codebase using this library, in order for this vulnerability to actually work?


When done with the vulnerability file, change the first line to CHECKED
git commit -m "replication done for ...."
Then QUIT IMMEDIATELY