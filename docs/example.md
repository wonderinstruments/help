---
title: "Git Rebase Workflow"
tags: [cli, git]
---

# Git Rebase Workflow

## Interactive Rebase

To rewrite recent commits:

    git rebase -i HEAD~3

This opens an editor where you can reorder, squash, or edit commits.

## Rebase onto main

To bring your branch up to date:

    git fetch origin
    git rebase origin/main

If conflicts arise, resolve them and run:

    git rebase --continue
