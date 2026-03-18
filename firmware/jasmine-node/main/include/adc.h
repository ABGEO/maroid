#pragma once

#include "esp_err.h"
#include <esp_adc/adc_oneshot.h>

esp_err_t setup_adc_oneshot_unit(adc_oneshot_unit_handle_t *ret_unit);
