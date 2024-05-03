#!/bin/bash

#------------------------------------------------------------------------------
# FUNCTIONS
#------------------------------------------------------------------------------
get_conn_string() {
	local -n _conn_string="${1}"

	local -r _admin_user=$($ords_cfg_cmd get --secret db.adminUser | tail -1)
	local -r _conn_type=$($ords_cfg_cmd get db.connectionType |tail -1)
	if [[ $_conn_type == "customurl" ]]; then
		local -r _conn=$($ords_cfg_cmd get db.customURL | tail -1)
	fi
	echo "Connection String (${_conn_type}): ${_conn}"

	_conn_string="${_admin_user%%/ *}/${secrets["dbadminusersecret"]}@${_conn} AS SYSDBA"
}

#------------------------------------------------------------------------------
function run_sql {
	local -r _conn_string="${1}"
	local -r _sql="${2}"
	local -n _output="${3}"
	local -i _rc=0
		
	# NOTE to maintainer; the heredoc must be TAB indented
	echo "Running SQL..."
	_output=$(cd ${APEX_HOME}/${APEX_VER} && sql -S /nolog <<-EOSQL
		WHENEVER SQLERROR EXIT 1
		WHENEVER OSERROR EXIT 1
		connect $_conn_string
		set serveroutput on echo off pause off feedback off
		set heading off wrap off linesize 1000 pagesize 0
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

#------------------------------------------------------------------------------
set_secret() {
	local _pool_name="${1}"
	local _config_key="${2}"
	local _config_val="${3}"
	local -i _rc=0

	if [ -n "${_config_val}" ]; then
		ords --config "$ORDS_CONFIG" config --db-pool "${_pool_name}" secret --password-stdin "${_config_key}" <<< "${_config_val}"
		rc=$?
		echo "Secret for ${_config_key} in pool ${_pool_name} set"
	else
		echo "Secret for ${_config_key} in pool ${_pool_name}, not defined"
	fi
}

#------------------------------------------------------------------------------
ords_upgrade() {
	local -r _pool_name="${1}"
	local -r _upgrade_key="${2}"
	local -i _rc=0
		
	if [[ -n "${secrets["dbadminusersecret"]}" ]]; then
		# Get usernames
		local -r ords_user=$($ords_cfg_cmd get db.username | tail -1)
		local -r ords_admin=$($ords_cfg_cmd get db.adminUser | tail -1)

		echo "Performing ORDS install/upgrade as $ords_admin into $ords_user on pool ${_pool_name}"
		ords --config "$ORDS_CONFIG" install --db-pool "${_pool_name}" --db-only \
			--admin-user "$ords_admin" --password-stdin <<< "${secrets["dbadminusersecret"]}"
		_rc=$?

		# Dar be bugs below deck with --db-user so using the above
		# ords --config "$ORDS_CONFIG" install --db-pool "$1" --db-only \
		# 	--admin-user "$ords_admin" --db-user "$ords_user" --password-stdin <<< "${!2}"
	fi

	return $_rc
}

#------------------------------------------------------------------------------
function get_apex_version() {
	local -r _conn_string="${1}"
	local -n _action="${2}"
	local -i _rc=0

	local -r _ver_sql="SELECT VERSION FROM DBA_REGISTRY WHERE COMP_ID='APEX';"
	run_sql "${_conn_string}" "${_ver_sql}" "_db_apex_version"
	_rc=$?

	if (( $_rc > 0 )); then
		echo "FATAL: Unable to connect to ${_conn_string} to get APEX version"
		return $_rc
	fi

	local -r _db_apex_version=${_db_apex_version//[^0-9.]/}
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

apex_upgrade() {
	local -r _conn_string="${1}"
	local -r _upgrade_key="${2}"
	local -i _rc=0

	if [[ -f ${APEX_HOME}/${APEX_VER}/apexins.sql ]] && [[ "${!_upgrade_key}" = "true" ]]; then
		echo "Starting Installation of APEX ${APEX_VER}"
		local -r _install_sql="@apexins SYSAUX SYSAUX TEMP /i/
			@apex_rest_config_core.sql /opt/oracle/apex/$APEX_VER/ ${secrets["dbsecret"]} ${secrets["dbsecret"]}"

		run_sql "${_conn_string}" "${_install_sql}" "_install_output"
		_rc=$?

		echo "Installation Output: ${_install_output}"
	fi

	return $_rc
}

apex_password() {
	local -r _conn_string="${1}"
	local -i _rc=0

	sed '/^accept/d' ${APEX_HOME}/${APEX_VER}/apxchpwd.sql > ${APEX_HOME}/${APEX_VER}/apxchpwdnew.sql
	echo "Setting APEX password"
	local -r _password_sql="
		exec APEX_INSTANCE_ADMIN.SET_PARAMETER( 'STRONG_SITE_ADMIN_PASSWORD', 'N' );
		DEFINE USERNAME=\"ADMIN\"
		DEFINE EMAIL=\"noreply@oracle.com\"
		DEFINE PASSWORD=\"${secrets["dbsecret"]}\"
		@apxchpwdnew.sql
		exec APEX_INSTANCE_ADMIN.SET_PARAMETER( 'STRONG_SITE_ADMIN_PASSWORD', 'Y' );
		DECLARE 
			l_user_id NUMBER;
		BEGIN
			APEX_UTIL.set_workspace(p_workspace => 'INTERNAL');
			l_user_id := APEX_UTIL.GET_USER_ID('ADMIN');
			APEX_UTIL.EDIT_USER(p_user_id => l_user_id, p_user_name  => 'ADMIN', p_change_password_on_first_use => 'Y');
		END;
		/"

	echo "${_password_sql}"
	run_sql "${_conn_string}" "${_password_sql}" "_db_apex_version"
	_rc=$?

	return ${_rc}
}

#------------------------------------------------------------------------------
# INIT
#------------------------------------------------------------------------------
declare -A pool_exit
for pool in "$ORDS_CONFIG"/databases/*; do
		pool_name=$(basename "$pool")
		pool_exit[${pool_name}]=0
		ords_cfg_cmd="ords --config $ORDS_CONFIG config --db-pool ${pool_name}"
		echo "Found Pool: $pool_name..."

		declare -A secrets
		for key in dbsecret dbadminusersecret dbcdbadminusersecret; do
				var_key="${pool_name}_${key}"
				var_val="${!var_key}"
				secrets[${key}]="${var_val}"
		done

		# Set Secrets
		set_secret "${pool_name}" "db.password" "${secrets["dbsecret"]}"
		rc=$((rc + $?))
		set_secret "${pool_name}" "db.adminUser.password" "${secrets["dbadminusersecret"]}"
		rc=$((rc + $?))
		set_secret "${pool_name}" "db.cdb.adminUser.password" "${secrets["dbcdbadminusersecret"]}"
		rc=$((rc + $?))

		if (( $rc > 0 )); then
			echo "FATAL: Unable to set secrets for pool ${pool_name}"
			pool_exit[${pool_name}]=1
			continue
		fi

		# ORDS Upgrade
		declare -r ords_upgrade_var=${pool_name}_autoupgrade_ords
		if [[ ${!ords_upgrade_var} != "true" ]]; then
			echo "ORDS Install/Upgrade not requested for ${_pool_name}"
			continue
		fi

		ords_upgrade "${pool_name}" "${pool_name}_autoupgrade_ords"
		rc=$?
		if (( $rc > 0 )); then
			echo "FATAL: Unable to preform requested ORDS install/upgrade on ${pool_name}"
			pool_exit[${pool_name}]=1
			continue
		fi

		# APEX Upgrade
		echo "---------------------------------------------------"
		declare -r apex_upgrade_var=${pool_name}_autoupgrade_apex
		if [[ ${!apex_upgrade_var} != "true" ]]; then
			echo "APEX Install/Upgrade not requested for ${pool_name}"
			continue
		fi

		get_conn_string "conn_string"
		if [[ -z ${conn_string} ]]; then
			echo "FATAL: Unable to get ${pool_name} database connect string"
			pool_exit[${pool_name}]=1
			continue
		fi

		get_apex_version "${conn_string}" "action"
		if [[ -z ${action} ]]; then
			echo "FATAL: Unable to get ${pool_name} APEX Version"
			pool_exit[${pool_name}]=1
			continue
		fi

		if [[ ${action} != "none" ]]; then
			apex_upgrade "${conn_string}" "${pool_name}_autoupgrade_apex"
			if (( $? > 0 )); then
				echo "FATAL: Unable to ${action} APEX for ${pool_name}"
				pool_exit[${pool_name}]=1
				continue
			fi			
		fi

		if [[ ${action} == "install" ]]; then
			apex_password "${conn_string}"
			if (( $? > 0 )); then
				echo "FATAL: Unable to update password for APEX for ${pool_name}"
				pool_exit[${pool_name}]=1
				continue
			fi			
		fi
done

for key in "${!pool_exit[@]}"; do
    echo "Pool: $key, Exit Code: ${pool_exit[$key]}"
		if (( ${pool_exit[$key]} > 0 )); then
			rc=1
		fi
done

exit $rc