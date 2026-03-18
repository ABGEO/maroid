#include "nvs_flash.h"

esp_err_t nvs_setup(void) {
  esp_err_t ret = nvs_flash_init();
  if (ret == ESP_ERR_NVS_NO_FREE_PAGES ||
      ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
    ret = nvs_flash_erase();
    if (ret != ESP_OK) {
      return ret;
    }

    return nvs_flash_init();
  }

  return ret;
}
