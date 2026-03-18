#pragma once

#include "esp_err.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"

/**
 * Wait for specific bits in an event group with a timeout.
 * Returns ESP_OK if all requested bits were set, ESP_ERR_TIMEOUT otherwise.
 */
static inline esp_err_t event_wait(EventGroupHandle_t group,
                                   EventBits_t bits_to_wait,
                                   TickType_t timeout_ticks, const char *tag) {
  EventBits_t bits =
      xEventGroupWaitBits(group, bits_to_wait, pdFALSE, pdTRUE, timeout_ticks);
  if ((bits & bits_to_wait) == bits_to_wait) {
    return ESP_OK;
  }

  ESP_LOGE(tag, "event_wait timed out (wanted 0x%lx, got 0x%lx)",
           (unsigned long)bits_to_wait, (unsigned long)bits);

  return ESP_ERR_TIMEOUT;
}
