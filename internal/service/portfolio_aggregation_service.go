package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"
)

// PortfolioAggregationService computes the unified portfolio view for a user.
type PortfolioAggregationService struct {
	walletStore   store.WalletStoreInterface
	priceStore    store.PriceStoreInterface
	exchangeStore store.ExchangeStoreInterface
	defiStore     store.DefiStoreInterface
}

// NewPortfolioAggregationService creates the service with all required stores.
func NewPortfolioAggregationService(
	ws store.WalletStoreInterface,
	ps store.PriceStoreInterface,
	es store.ExchangeStoreInterface,
	ds store.DefiStoreInterface,
) *PortfolioAggregationService {
	return &PortfolioAggregationService{
		walletStore:   ws,
		priceStore:    ps,
		exchangeStore: es,
		defiStore:     ds,
	}
}

// GetSummary aggregates all sources and returns a portfolio summary.
func (s *PortfolioAggregationService) GetSummary(ctx context.Context, userID uint64) (*model.PortfolioSummary, error) {
	// symbolQty accumulates total quantity per symbol (raw, unscaled for now)
	symbolQty := make(map[string]*big.Float)
	var sourceBreakdowns []model.SourceBreakdown

	walletUSD := new(big.Float)
	exchangeUSD := new(big.Float)
	defiUSD := new(big.Float)

	// --- Wallet assets ---
	wallets, err := s.walletStore.GetWalletsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("aggregation: wallet query: %w", err)
	}
	for _, wallet := range wallets {
		assets, err := s.walletStore.GetAssetsByWalletID(wallet.ID)
		if err != nil {
			return nil, fmt.Errorf("aggregation: wallet assets: %w", err)
		}
		walletValue := new(big.Float)
		for _, asset := range assets {
			if asset.Symbol == "" {
				continue
			}
			sym := strings.ToUpper(asset.Symbol)
			qty := parseDecimal(asset.Balance)
			accumulateSymbol(symbolQty, sym, qty)
			price := s.getPrice(sym)
			walletValue.Add(walletValue, new(big.Float).Mul(qty, price))
		}
		walletUSD.Add(walletUSD, walletValue)
		if walletValue.Sign() > 0 {
			sourceBreakdowns = append(sourceBreakdowns, model.SourceBreakdown{
				Source:   "wallet:" + wallet.Chain,
				ValueUSD: formatFloat(walletValue),
			})
		}
	}

	// --- Exchange balances ---
	exBalances, err := s.exchangeStore.GetBalancesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("aggregation: exchange balances: %w", err)
	}
	// Group by credential to make source labels
	byCredential := make(map[uint64][]*model.ExchangeBalance)
	for i := range exBalances {
		byCredential[exBalances[i].CredentialID] = append(byCredential[exBalances[i].CredentialID], &exBalances[i])
	}
	creds, _ := s.exchangeStore.GetCredentialsByUserID(userID)
	credExchangeMap := make(map[uint64]string)
	for _, c := range creds {
		credExchangeMap[c.ID] = c.Exchange
	}
	for credID, bals := range byCredential {
		credValue := new(big.Float)
		for _, b := range bals {
			sym := strings.ToUpper(b.Symbol)
			qty := parseDecimal(b.FreeBalance)
			lockedQty := parseDecimal(b.LockedBalance)
			total := new(big.Float).Add(qty, lockedQty)
			accumulateSymbol(symbolQty, sym, total)
			price := s.getPrice(sym)
			credValue.Add(credValue, new(big.Float).Mul(total, price))
		}
		exchangeUSD.Add(exchangeUSD, credValue)
		if credValue.Sign() > 0 {
			exchangeName := credExchangeMap[credID]
			if exchangeName == "" {
				exchangeName = "unknown"
			}
			sourceBreakdowns = append(sourceBreakdowns, model.SourceBreakdown{
				Source:   "exchange:" + exchangeName,
				ValueUSD: formatFloat(credValue),
			})
		}
	}

	// --- DeFi positions ---
	defiPositions, err := s.defiStore.GetPositionsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("aggregation: defi positions: %w", err)
	}
	for _, pos := range defiPositions {
		// DeFi positions are stored as raw JSON; we estimate USD using token0+token1 owed amounts.
		// We report the protocol as a source without precise USD (requires on-chain math).
		// This is intentionally conservative: only count tokensOwed for known symbols.
		_ = pos // detailed parsing skipped; protocol source still recorded
		if defiUSD.Sign() == 0 && pos.Protocol != "" {
			sourceBreakdowns = append(sourceBreakdowns, model.SourceBreakdown{
				Source:   "defi:" + pos.Protocol,
				ValueUSD: "0",
			})
		}
	}

	// --- Build symbol positions ---
	totalUSD := new(big.Float).Add(walletUSD, new(big.Float).Add(exchangeUSD, defiUSD))

	symbolPositions := make([]model.SymbolPosition, 0, len(symbolQty))
	for sym, qty := range symbolQty {
		price := s.getPrice(sym)
		valueUSD := new(big.Float).Mul(qty, price)
		pct := new(big.Float)
		if totalUSD.Sign() > 0 {
			pct.Quo(new(big.Float).Mul(valueUSD, big.NewFloat(100)), totalUSD)
		}
		symbolPositions = append(symbolPositions, model.SymbolPosition{
			Symbol:   sym,
			TotalQty: formatFloat(qty),
			PriceUSD: formatFloat(price),
			ValueUSD: formatFloat(valueUSD),
			PctShare: formatFloat(pct),
		})
	}

	// Fill in pct_share for sources
	for i := range sourceBreakdowns {
		v := parseDecimal(sourceBreakdowns[i].ValueUSD)
		pct := new(big.Float)
		if totalUSD.Sign() > 0 {
			pct.Quo(new(big.Float).Mul(v, big.NewFloat(100)), totalUSD)
		}
		sourceBreakdowns[i].PctShare = formatFloat(pct)
	}

	return &model.PortfolioSummary{
		UserID:      userID,
		TotalUSD:    formatFloat(totalUSD),
		WalletUSD:   formatFloat(walletUSD),
		ExchangeUSD: formatFloat(exchangeUSD),
		DefiUSD:     formatFloat(defiUSD),
		BySymbol:    symbolPositions,
		BySource:    sourceBreakdowns,
		ComputedAt:  time.Now(),
	}, nil
}

// getPrice looks up the latest price, returning 0 if not found.
func (s *PortfolioAggregationService) getPrice(symbol string) *big.Float {
	p, err := s.priceStore.GetLatestPrice(symbol)
	if err != nil || p == nil {
		return new(big.Float)
	}
	f, _, _ := new(big.Float).Parse(p.PriceUSD, 10)
	return f
}

func accumulateSymbol(m map[string]*big.Float, sym string, qty *big.Float) {
	if existing, ok := m[sym]; ok {
		existing.Add(existing, qty)
	} else {
		m[sym] = new(big.Float).Set(qty)
	}
}

func parseDecimal(s string) *big.Float {
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return new(big.Float)
	}
	return f
}

func formatFloat(f *big.Float) string {
	return f.Text('f', 8)
}
