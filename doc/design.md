# Design: Library Type Identification in google-cloud-python

This document outlines the logic for determining whether a client library within the `google-cloud-python` repository is Generated, Handwritten, or Hybrid.

## Core Principle

The source of truth for identifying auto-generated files is a file named `state.yaml` located within the `google-cloud-python` repository. This file contains a list of regular expression patterns under a `remove_regex` key.

Any file whose path matches one of these patterns is considered **generated**. Any file that does not match is considered **handwritten**.

## Library Type Definitions

Based on this principle, we can classify each library as follows:

### 1. Pure GAPIC (Generated)

A library is considered "Pure GAPIC" or fully generated if **all** of its files match the patterns specified in `state.yaml`. There is no handwritten code in the library's directory.

### 2. Handwritten

A library is considered "Handwritten" if **none** of its files match the patterns in `state.yaml`. The entire library is manually coded and maintained.

### 3. Hybrid

A library is considered "Hybrid" if it contains a **mix** of files that match the patterns in `state.yaml` and files that do not. This indicates that the library uses a base of generated code but is supplemented with custom, handwritten code.
