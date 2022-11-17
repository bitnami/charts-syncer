# Examples

This directory contains examples that show the various ways to use the Asset Relocation Tool for Kubernetes.

* [Simple Chart](simple-chart) shows relocating a Helm chart
* [Chart with Subcharts](chart-with-subcharts) shows relocating a Helm chart that contains subcharts
* [Concourse Pipeline](concourse-pipeline) shows how to use the tool within a CI/CD tool like [Concourse](https://concourse-ci.org/)

### Running the examples

This repository does not store the input charts, but uses [vendir](https://carvel.dev/vendir/) to fetch them.
To run the examples, use `vendir sync` in this directory.
