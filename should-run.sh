#!/bin/bash

trigger_commit=$1
service_name=$2

echo "Interest service: $service_name, commit: $trigger_commit"

force_run=$(git show -s --format=%s $trigger_commit | grep -q -F "[force]" && echo true || echo false)

services_list=()

if [[ "$force_run" == "true" ]]; then
  echo "Detected [force] message"
  services_list=( $(ls -1) )
else
  echo "Going to determine changed dirs"
  parent_commits=()
  parent_commits=( $(git rev-parse $trigger_commit^@) )

  echo "Going to examine parent commits"
  changed_dirs=()
  for parent_commit in "${parent_commits[@]}"; do
      echo "Examining parent commit ${parent_commit}"
      local_changed_dirs=( $(git diff --dirstat=files,0 ${parent_commit} ${trigger_commit} | sed 's/^[ 0-9.]\+% //g' | cut -d'/' -f1 | uniq) )
      echo "in parent commit ${parent_commit} there are following changed dirs"
      for changed_dir in "${local_changed_dirs[@]}"; do
          echo "-> ${changed_dir}"
          changed_dirs+=(${changed_dir})
      done
  done

  services_list=($(printf "%s\n" "${changed_dirs[@]}" | sort -u))
fi

echo "List of changed services: ${services_list[@]}"
#echo "${services_list[@]}"

echo "${services_list[@]}" | grep -E -q ${service_name}
