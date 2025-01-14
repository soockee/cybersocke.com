---
name: "GCP Project"
slug: "gcp-label-project"
tags: ["cloud", "storage", "distributed systems"]
date: 2025-01-14
description: "Entry-Level project to get to know the cloud."
---
# Resource Labeling in GCP: Understanding Software Development Key Concepts
## Introduction

In this project, we will gradually build the foundational knowledge needed to implement a service-specific module in Google Cloud Platform (GCP). 
The goal is to develop a module that analyzes the resources of a selected GCP service, lists their labels, and applies new labels based on the specific requirements of the service.

Our initial focus will be on learning the necessary tools and concepts before diving into the actual implementation. 
This step-by-step approach ensures that we are well-prepared and have a basic understanding of all involved parts of the software development lifecycle from the ground up.

## Step-by-Step Project Plan

### Introduction to the Go Programming Language
Goal: Gain a basic understanding of Go to effectively use GCP APIs in the later stages.

Topics:

Setting up the Go environment and development tools (e.g., Visual Studio Code).
Writing a simple “Hello, World!” program.
Learning basic concepts like variables, loops, and functions.
Resources:
- [Official Go Documentation](https://go.dev/doc/)
- [Tour of Go](https://go.dev/tour/welcome/1)

### Introduction to Git
Goal: Understand version control and gain hands-on experience with Git workflows.
Topics:

- Installing and setting up Git.
- Initializing a local repository.
- Versioning the “Hello, World!” project:
    - Create and add files (git add).
    - Commit changes (git commit).
    - Track changes with Git logs (git log).
- Connecting to a remote repository (e.g., GitHub).
- Practice: Upload the “Hello, World!” project to a personal repository on GitHub.

### Introduction to GitHub CI/CD
Goal: Learn the basics of Continuous Integration and Deployment (CI/CD).

Topics:

- Understanding GitHub Actions and their role in automation.
- Creating a simple workflow:
    - Automate the testing of the “Hello, World!” project.
    - Build a pipeline script within the .github/workflows directory structure.
- Outcome: Successfully run a workflow that ensures the code is built and tested.
- Resources:
    - [github doc](https://docs.github.com/en/actions)

#  Deployment of a GitHub Pages Project

Goal: Create a GitHub Pages site to document the learning process.

Topics:

- Setting up a new GitHub repository for the project.
- Enabling GitHub Pages in the repository settings.
- Creating content for the page:
    - Notes on Go basics.
    - Explanation of CI/CD concepts and workflows.
    - Tips on Git and GitHub usage.
    - Insights into GCP tools and modules.
- Writing the site using Markdown or HTML/CSS.
- Automating deployment with a GitHub Actions workflow.
- Outcome: A functional GitHub Pages website showcasing your learning journey.
- Resources:
    - GitHub Pages Documentation
    - GitHub Actions for Pages Deployment

# GCP Project: Implementing a Service-Specific Module
Goal: Develop a module to manage labels for a GCP service.

Topics:

Analyzing the API documentation for the Cloud Run Admin API.
Implementing a function to list all resources that support labels (list).
Developing a function to set or update labels (patch).
Integrating these functions into the existing architecture.

Note:
- use v2 of the API

Resources:
- internal repo ref
[GCP API Documentation](https://cloud.google.com/run/docs/reference/rest).

## Sources

- todo ☺️ collect some stuff
