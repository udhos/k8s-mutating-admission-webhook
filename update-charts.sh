#!/bin/bash

# the ./docs dir is published as https://udhos.github.io/k8s-mutating-admission-webhook/

chart_dir=charts/k8s-mutating-admission-webhook

# generate new chart package from source into ./docs
helm package $chart_dir -d ./docs

#
# copy new chart into ./charts-tmp
#

chart_name=$(gojq --yaml-input -r .name < $chart_dir/Chart.yaml)
chart_version=$(gojq --yaml-input -r .version < $chart_dir/Chart.yaml)
chart_pkg=${chart_name}-${chart_version}.tgz
rm -rf charts-tmp
mkdir -p charts-tmp
cp docs/${chart_pkg} charts-tmp

#
# merge new chart index into docs/index.yaml
#

git checkout docs/index.yaml ;# reset index
helm repo index charts-tmp --url https://udhos.github.io/k8s-mutating-admission-webhook/ --merge docs/index.yaml
# new merged chart index was generated as ./charts-tmp/index.yaml,
# copy it back to ./docs
cp charts-tmp/index.yaml docs

# regenerate the index from existing chart packages
#helm repo index ./docs --url https://udhos.github.io/k8s-mutating-admission-webhook/

echo "#"
echo "# check that ./docs is fine then:"
echo "#"
echo "git add docs"
echo "git commit -m 'Update chart repository.'"
echo "git push"
