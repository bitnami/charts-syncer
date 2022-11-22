# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause

.PHONY: build clean rebuild
build: wordpress/.unarchived
rebuild: clean build
clean:
	rm -rf wordpress

wordpress-11.0.4.tgz:
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm pull bitnami/wordpress --version 11.0.4

wordpress/.unarchived: wordpress-11.0.4.tgz
	tar zxvf wordpress-11.0.4.tgz
	touch wordpress/.unarchived

