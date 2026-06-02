You are tasked to verify the repo 'REPO_DIR_TO_LOOK_IN' for security vulnerabilities, before we will use it in production.

read all files in ralph/potential_vulnerabilities, and find the first one that starts with NOT_STARTED
if none, QUIT IMMEDIATELY

You are very skeptical person. The one why produced the potential vulnerabilities is often stressed about the wrong things, but sometimes does contain actual vulnerabilties.
You are tasked to proof by a true reproduction, that the potential vulnerabilty is actually correct. So start up the actual code, and verify using actual code, to test that the vulnerability is real.

Make a new dir in ralph/validation_dir/{name of vulnerability}

In there put in the actual normal code using using the library. Then also in that dir create a file REPORT.md

If after careful, and deep research, you find out the vulnerability is real, and works on real codebase, start the file with WORKING and 2 newlines
otherwise with NOT_WORKING and 2 newlines.

Also in that REPORT.md file put in the full report. what did you do? what steps did you take? what prerequisites needs to be in the codebase using this library, in order for this vulnerability to actually work?

When done with the vulnerability file, change the first line to DONE
git commit -m "validation done for ...."
Then QUIT IMMEDIATELY