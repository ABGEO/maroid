#pragma once

#include "esp_err.h"
#include "mqtt_client.h"

esp_err_t mqtt_start(esp_mqtt_client_handle_t *client);

esp_err_t mqtt_wait_published(void);

void mqtt_stop(esp_mqtt_client_handle_t client);

int mqtt_send_reading(esp_mqtt_client_handle_t client, const char *reading_type,
                      const char *reading);
