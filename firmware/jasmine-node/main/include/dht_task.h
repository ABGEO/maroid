#pragma once

#include "freertos/FreeRTOS.h"

#include <mqtt_client.h>

#define DHT_DONE_BIT BIT1
#define DHT_ERR_BIT BIT5

typedef struct {
  esp_mqtt_client_handle_t mqtt_client;
  EventGroupHandle_t event_group;
} dht_task_params_t;

void dht_task(void *arg);
