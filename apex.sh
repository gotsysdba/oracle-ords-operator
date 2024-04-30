#!/bin/bash

function run_sql {
	local -r _conn_string="${1}"
	local -r _sql="${2}"
	local -n _output="${3}"
	local -i _rc=0
		
	# NOTE to maintainer; the heredoc must be TAB indented
	_output=$(sql -S /nolog <<-EOSQL
		WHENEVER SQLERROR EXIT 1
		WHENEVER OSERROR EXIT 1
		connect $_conn_string
		set serveroutput on echo off
		set pause off feedback off
		set heading off wrap off
		set linesize 1000 pagesize 0
		SET TERMOUT OFF VERIFY OFF
		${_sql}
		exit;
		EOSQL
	)
	_rc=$?

	if (( $_rc > 0 )); then
		echo "${_output}"
	fi
	
	return $_rc
}

function compare_versions() {
    local db_ver=$1
    local im_ver=$2

    IFS='.' read -r -a db_ver_array <<< "$db_ver"
    IFS='.' read -r -a im_ver_array <<< "$im_ver"

    # Compare each component
    local i
    for i in "${!db_ver_array[@]}"; do
        if [[ "${db_ver_array[$i]}" -lt "${im_ver_array[$i]}" ]]; then
			# db_ver < im_ver (upgrade)
            return 0
        elif [[ "${db_ver_array[$i]}" -gt "${im_ver_array[$i]}" ]]; then
			# db_ver < im_ver (do nothing)
            return 1
        fi
    done
    # db_ver == im_ver (do nothing)
    return 1
}

function apex_install() {
	typeset -r _conn_string="${1}"
	typeset -r _password="${2}"
	typeset -i _rc=0

	if [ -f ${APEX_HOME}/${APEX_VER}/apexins.sql ]; then
		echo "Installing APEX ${APEX_VER}"
		declare -r _install_sql="@apexins SYSAUX SYSAUX TEMP /i/
			@apex_rest_config_core.sql /opt/oracle/apex/$APEX_VER/ ${_password} ${_password}"
		cd ${APEX_HOME}/${APEX_VER} && run_sql "${_conn_string}" "${_install_sql}" "_install_output"
		_rc=$?
	fi

	echo "${_install_output}"
	if (( $_rc > 0 )); then
		echo "FATAL: Unable to install APEX"
	fi
	
	return $_rc
}

function get_version() {
	declare -r _conn_string="${1}"
	declare -n _action="${2}"
	declare -i _rc=0

	declare -r _ver_sql="SELECT VERSION FROM DBA_REGISTRY WHERE COMP_ID='APEX';"
	run_sql "${_conn_string}" "${_ver_sql}" "_db_apex_version"
	_rc=$?

	if (( $_rc > 0 )); then
		echo "FATAL: Unable to connect to ${conn_string} to get APEX version"
		return $_rc
	fi

	_db_apex_version=${_db_apex_version//[^0-9.]/}
	echo "Database APEX Version: ${_db_apex_version:-Not Installed}"

	_action="none"
	if [[ -z "${_db_apex_version}" ]]; then
		echo "Installing APEX ${APEX_VER}"
		_action="install"
	elif compare_versions ${_db_apex_version} ${APEX_VER}; then
		echo "Upgrading from ${_db_apex_version} to ${APEX_VER}"
		_action="upgrade"
	else
		echo "No Installation/Upgrade Required"
	fi

	return $_rc
}

#------------------------------------------------------------------------------
# INIT
#------------------------------------------------------------------------------
declare -r pool_name=$1
if [[ -z $pool_name ]]; then
	echo "No Pool Provided"
	exit 1
fi

declare -r ords_cmd="ords --config $ORDS_CONFIG config --db-pool $pool_name"
declare -r admin_pass=$($ords_cmd get --secret db.adminUser.password |tail -1)
declare -r admin_user=$($ords_cmd get --secret db.adminUser | tail -1)
declare -r conn_type=$($ords_cmd get db.connectionType |tail -1)

if [[ $conn_type == "customurl" ]]; then
	declare -r connection=$($ords_cmd get db.customURL | tail -1)
fi
declare conn_string="${admin_user%%/ *}/${admin_pass}@${connection} AS SYSDBA"
echo "${conn_string}"
exit 1

get_version "${conn_string}" "action"
rc=$?

if (( $rc == 0 )) && [[ ${action} != "none" ]]; then
	apex_install "${conn_string}"
	rc=$?
fi

if (( $rc == 0 )) && [[ ${action} == "install" ]]; then
	apex_password "${conn_string}"
	rc=$?
fi

exit $rc