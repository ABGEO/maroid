#pragma once

#include "esp_err.h"
#include "mqtt_client.h"

esp_err_t sensors_run(esp_mqtt_client_handle_t mqtt_client);
