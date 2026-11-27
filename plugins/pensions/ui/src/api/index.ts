import type { PluginHost } from '@maroid/plugin-sdk';

import { createAssetValueApi } from './asset_value';
import { createConnectionApi } from './connection';
import { createContributionApi } from './contribution';
import { createPortfolioApi } from './portfolio';
import { createSummaryApi } from './summary';

export type {
	AssetValue,
	Connection,
	ConnectionState,
	Contribution,
	ContributionFilter,
	ContributionMonth,
	ContributionTotal,
	Employer,
	Holder,
	Money,
	PortfolioShare,
	SourceSums,
	Summary,
	SummaryView,
	UnitPrice
} from './types';

export function createPensionApi(host: PluginHost) {
	return {
		connection: createConnectionApi(host.api),
		summary: createSummaryApi(host.api),
		assetValues: createAssetValueApi(host.api),
		contributions: createContributionApi(host.api),
		portfolios: createPortfolioApi(host.api)
	};
}

export type PensionApi = ReturnType<typeof createPensionApi>;
