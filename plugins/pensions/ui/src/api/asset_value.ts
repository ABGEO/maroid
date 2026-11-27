import type { ApiClient, Page } from '@maroid/plugin-sdk';

import { PAGE_LIMIT, readAll } from '../lib/pages';
import type { AssetValue } from './types';

export function createAssetValueApi(api: ApiClient) {
	return {
		/** Reads every page of the asset values from a date YYYY-MM-DD, or every one with no date. */
		since: (from?: string): Promise<AssetValue[] | null> =>
			readAll((link) =>
				link
					? api.follow<Page<AssetValue>>(link)
					: api.get<Page<AssetValue>>('/asset-values', { params: { from, limit: PAGE_LIMIT } })
			)
	};
}
