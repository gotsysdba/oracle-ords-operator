#!/bin/bash

#------------------------------------------------------------------------------
# FUNCTIONS
#------------------------------------------------------------------------------
get_conn_string() {
	local -n _conn_string="${1}"

	local -r _admin_user=$($ords_cfg_cmd get --secret db.adminUser | tail -1)
	local _conn_type=$($ords_cfg_cmd get db.connectionType |tail -1)
	if [[ $_conn_type == "customurl" ]]; then
		local -r _conn=$($ords_cfg_cmd get db.customURL | tail -1)
	elif [[ $_conn_type == "tns" ]]; then
		local -r _tns_service=$($ords_cfg_cmd get db.tnsAliasName | tail -1)
		local -r _conn=${_tns_service}
	elif [[ $_conn_type == "basic" ]]; then
		local -r _host=$($ords_cfg_cmd get db.hostname | tail -1)
		local -r _port=$($ords_cfg_cmd get db.port | tail -1)
		local -r _service=$($ords_cfg_cmd get db.servicename | tail -1)
		local -r _sid=$($ords_cfg_cmd get db.sid | tail -1)

		if [[ -n ${_host} ]] && [[ -n ${_port} ]]; then
			if [[ -n ${_service} ]] || [[ -n ${_sid} ]]; then
				local -r _conn=${_host}:${_port}/${_service:-$_sid}
			fi
		fi
	else 
		# wallet
		_conn_type="wallet"
		local -r _wallet_service=$($ords_cfg_cmd get db.wallet.zip.service | tail -1)
		local -r _conn=${_wallet_service}
	fi

	if [[ -n ${_conn} ]]; then
		echo "Connection String (${_conn_type}): ${_conn}"
		_conn_string="${_admin_user%%/ *}/${config["dbadminusersecret"]}@${_conn}"
		if [[ ${_admin_user%%/ *} == "SYS" ]]; then
			_conn_string="${_conn_string=} AS SYSDBA"
		fi
	fi
}

#------------------------------------------------------------------------------
function run_sql {
	local -r _conn_string="${1}"
	local -r _sql="${2}"
	local -n _output="${3}"
	local -i _rc=0
	
	if [[ -z ${_sql} ]]; then
		echo "FATAL: Dear Developer.. you've got a bug calling run_sql" && exit 1
	fi
	## Get TNS_ADMIN location
	local -r _tns_admin=$($ords_cfg_cmd get db.tnsDirectory | tail -1)
	if [[ ! $_tns_admin =~ "Cannot get setting" ]]; then
		echo "Setting: TNS_ADMIN=${_tns_admin}"
		export TNS_ADMIN=${_tns_admin}
	fi

	## Get ADB Wallet
	local -r _wallet_zip_path=$($ords_cfg_cmd get db.wallet.zip.path | tail -1)
	if [[ ! $_wallet_zip_path =~ "Cannot get setting" ]]; then
		echo "Using: set cloudconfig ${_wallet_zip_path}"
		local -r _cloudconfig="set cloudconfig ${_wallet_zip_path}"
	fi

	# NOTE to maintainer; the heredoc must be TAB indented
	echo "Running SQL..."
	#_output=$(cd ${APEX_HOME}/${APEX_VER} && sql -S /nolog <<-EOSQL
	_output=$(cd ${APEX_HOME}/${APEX_VER} && sql -S -nohistory -noupdates /nolog <<-EOSQL
		WHENEVER SQLERROR EXIT 1
		WHENEVER OSERROR EXIT 1
		${_cloudconfig}
		connect $_conn_string
		set serveroutput on echo off pause off feedback off
		set heading off wrap off linesize 1000 pagesize 0
		SET TERMOUT OFF VERIFY OFF
		${_sql}
		exit;
		EOSQL
	)
	_rc=$?

	if (( ${_rc} > 0 )); then
		echo "SQLERROR: ${_output}"
	fi
	
	return $_rc
}

#------------------------------------------------------------------------------
function check_adb() {
	local -r _conn_string=$1
	local -n _is_adb=$2

	local -r _adb_chk_sql="
		SELECT COUNT(*) FROM (
			SELECT JSON_VALUE(cloud_identity, '\$.DATABASE_OCID') AS database_ocid 
			FROM v\$pdbs) t
		WHERE t.database_ocid like '%AUTONOMOUS%';"
	echo "Checking if Database is an ADB"
	run_sql "${_conn_string}" "${_adb_chk_sql}" "_adb_check"
	_rc=$?

	if (( ${_rc} == 0 )); then
		_adb_check=${_adb_check//[[:space:]]/}
		echo "ADB Check: ${_adb_check}"
		if (( ${_adb_check} == 1 )); then
			_is_adb=${_adb_check//[[:space:]]/}
		fi
	fi

	return ${_rc}
}

function create_adb_user() {
	local -r _conn_string="${1}"
	local -r _pool_name="${2}"
                        
	local _config_user=$($ords_cfg_cmd get db.username | tail -1)

	if [[ -z ${_config_user} ]] || [[ ${_config_user} == "ORDS_PUBLIC_USER" ]]; then
		echo "FATAL: You must specify a db.username <> ORDS_PUBLIC_USER in pool ${_pool_name}"
		return 1
	fi

	local -r _adb_user_sql="
    DECLARE
      l_user VARCHAR2(255);
      l_cdn  VARCHAR2(255);
    BEGIN
      BEGIN
        SELECT USERNAME INTO l_user FROM DBA_USERS WHERE USERNAME='${_config_user}';
        EXECUTE IMMEDIATE 'ALTER USER \"${_config_user}\" PROFILE ORA_APP_PROFILE';
        EXECUTE IMMEDIATE 'ALTER USER \"${_config_user}\" IDENTIFIED BY \"${config["dbsecret"]}\"';
		DBMS_OUTPUT.PUT_LINE('${_config_user} Exists - Password reset');
      EXCEPTION
        WHEN NO_DATA_FOUND THEN
          EXECUTE IMMEDIATE 'CREATE USER \"${_config_user}\" IDENTIFIED BY \"${config["dbsecret"]}\" PROFILE ORA_APP_PROFILE';
		  DBMS_OUTPUT.PUT_LINE('${_config_user} Created');
      END;
      EXECUTE IMMEDIATE 'GRANT CONNECT TO \"${_config_user}\"';
      BEGIN
        SELECT USERNAME INTO l_user FROM DBA_USERS WHERE USERNAME='ORDS_PLSQL_GATEWAY_OPER';
          EXECUTE IMMEDIATE 'ALTER USER \"ORDS_PLSQL_GATEWAY_OPER\" PROFILE DEFAULT';
          EXECUTE IMMEDIATE 'ALTER USER \"ORDS_PLSQL_GATEWAY_OPER\" NO AUTHENTICATION';
		  DBMS_OUTPUT.PUT_LINE('ORDS_PLSQL_GATEWAY_OPER Exists');
        EXCEPTION
          WHEN NO_DATA_FOUND THEN
            EXECUTE IMMEDIATE 'CREATE USER \"ORDS_PLSQL_GATEWAY_OPER\" NO AUTHENTICATION PROFILE DEFAULT';
			DBMS_OUTPUT.PUT_LINE('ORDS_PLSQL_GATEWAY_OPER Created');
      END;
      EXECUTE IMMEDIATE 'GRANT CONNECT TO \"ORDS_PLSQL_GATEWAY_OPER\"';
      EXECUTE IMMEDIATE 'ALTER USER \"ORDS_PLSQL_GATEWAY_OPER\" GRANT CONNECT THROUGH \"${_config_user}\"';
      ORDS_ADMIN.PROVISION_RUNTIME_ROLE (
          p_user => '${_config_user}'
        ,p_proxy_enabled_schemas => TRUE
      );
      ORDS_ADMIN.CONFIG_PLSQL_GATEWAY (
          p_runtime_user => '${_config_user}'
        ,p_plsql_gateway_user => 'ORDS_PLSQL_GATEWAY_OPER'
      );
	  -- TODO: Only do this if ADB APEX Version <> this ORDS Version
      BEGIN
        SELECT images_version INTO L_CDN
          FROM APEX_PATCHES
        where is_bundle_patch = 'Yes'
        order by patch_version desc
        fetch first 1 rows only;
      EXCEPTION WHEN NO_DATA_FOUND THEN
        select version_no INTO L_CDN
          from APEX_RELEASE;
      END;
      apex_instance_admin.set_parameter(
          p_parameter => 'IMAGE_PREFIX',
          p_value     => 'https://static.oracle.com/cdn/apex/'||L_CDN||'/'
      );
    END;
	/"

	run_sql "${_conn_string}" "${_adb_user_sql}" "_adb_user_sql_output"
	_rc=$?

	echo "Installation Output: ${_adb_user_sql_output}"
	return ${_rc}
}

#------------------------------------------------------------------------------
function compare_versions() {
	local _db_ver=$1
	local _im_ver=$2

	IFS='.' read -r -a _db_ver_array <<< "$_db_ver"
	IFS='.' read -r -a _im_ver_array <<< "$_im_ver"

	# Compare each component
	local i
	for i in "${!_db_ver_array[@]}"; do
		if [[ "${_db_ver_array[$i]}" -lt "${_im_ver_array[$i]}" ]]; then
		# _db_ver < _im_ver (upgrade)
			return 0
		elif [[ "${_db_ver_array[$i]}" -gt "${_im_ver_array[$i]}" ]]; then
		# _db_ver < _im_ver (do nothing)
			return 1
		fi
	done
	# _db_ver == __im_ver (do nothing)
	return 1
}

#------------------------------------------------------------------------------
set_secret() {
	local -r _pool_name="${1}"
	local -r _config_key="${2}"
	local -r _config_val="${3}"
	local -i _rc=0

	if [[ -n "${_config_val}" ]]; then
		ords --config "$ORDS_CONFIG" config --db-pool "${_pool_name}" secret --password-stdin "${_config_key}" <<< "${_config_val}"
		_rc=$?
		echo "${_config_key} in pool ${_pool_name} set"
	else
		echo "${_config_key} in pool ${_pool_name}, not defined"
		_rc=0
	fi

	return ${_rc}
}

#------------------------------------------------------------------------------
ords_upgrade() {
	local -r _pool_name="${1}"
	local -r _upgrade_key="${2}"
	local -i _rc=0
		
	if [[ -n "${config["dbadminusersecret"]}" ]]; then
		# Get usernames
		local -r ords_user=$($ords_cfg_cmd get db.username | tail -1)
		local -r ords_admin=$($ords_cfg_cmd get db.adminUser | tail -1)

		echo "Performing ORDS install/upgrade as $ords_admin into $ords_user on pool ${_pool_name}"
		ords --config "$ORDS_CONFIG" install --db-pool "${_pool_name}" --db-only \
			--admin-user "$ords_admin" --password-stdin <<< "${config["dbadminusersecret"]}"
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
			@apex_rest_config_core.sql /opt/oracle/apex/$APEX_VER/ ${config["dbsecret"]} ${config["dbsecret"]}"

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
		DEFINE PASSWORD=\"${config["dbsecret"]}\"
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
	rc=0
	pool_name=$(basename "$pool")
	pool_exit[${pool_name}]=0
	ords_cfg_cmd="ords --config $ORDS_CONFIG config --db-pool ${pool_name}"
	echo "Found Pool: $pool_name..."

	declare -A config
	for key in dbsecret dbadminusersecret dbcdbadminusersecret dbwalletzip dbtnsdirectory; do
			var_key="${pool_name}_${key}"
			var_val="${!var_key}"
			config[${key}]="${var_val}"
	done

	# Set Secrets
	set_secret "${pool_name}" "db.password" "${config["dbsecret"]}"  "true"
	rc=$((rc + $?))
	set_secret "${pool_name}" "db.adminUser.password" "${config["dbadminusersecret"]}" "true"
	rc=$((rc + $?))
	set_secret "${pool_name}" "db.cdb.adminUser.password" "${config["dbcdbadminusersecret"]}" "true"
	rc=$((rc + $?))

	if (( ${rc} > 0 )); then
		echo "FATAL: Unable to set configuration for pool ${pool_name}"
		pool_exit[${pool_name}]=1
		continue
	elif [[ -z ${config["dbsecret"]} ]]; then
		echo "FATAL: db.password must be specified for ${pool_name}"
		pool_exit[${pool_name}]=1
		continue
	elif [[ -z ${config["dbadminusersecret"]} ]]; then
		echo "INFO: No additional configuration for ${pool_name}"
		continue
	fi

	get_conn_string "conn_string"
	if [[ -z ${conn_string} ]]; then
		echo "FATAL: Unable to get ${pool_name} database connect string"
		pool_exit[${pool_name}]=1
		continue
	fi

	check_adb "${conn_string}" "is_adb"
	rc=$?
	if (( ${rc} > 0 )); then
		pool_exit[${pool_name}]=1
		continue
	fi

	if (( is_adb )); then
		# Create ORDS User
		echo "Processing ADB in Pool: ${pool_name}"
		create_adb_user "${conn_string}" "${pool_name}"
	else
		# ORDS Upgrade
		declare -r ords_upgrade_var=${pool_name}_autoupgrade_ords
		if [[ ${!ords_upgrade_var} != "true" ]]; then
			echo "ORDS Install/Upgrade not requested for ${pool_name}"
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
	fi
done

for key in "${!pool_exit[@]}"; do
    echo "Pool: $key, Exit Code: ${pool_exit[$key]}"
	if (( ${pool_exit[$key]} > 0 )); then
		rc=1
	fi
done

exit $rc