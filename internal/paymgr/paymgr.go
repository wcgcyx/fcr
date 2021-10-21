package paymgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"

	"github.com/wcgcyx/fcr/internal/payoffer"
)

// PaymentManager interacts with blockchain to create/topup payment channels.
// It also maintains the state of all channels, provide interface to pay and to receive payments.
type PaymentManager interface {
	// GetPrvKey gets the private key associated of given currency id.
	// It takes the currency id as the argument.
	// It returns the private key bytes and error if not existed.
	GetPrvKey(currencyID uint64) ([]byte, error)

	// GetRootAddress gets the root address of given currency id.
	// It takes a currency id as the argument.
	// It returns the root address and error if not existed.
	GetRootAddress(currencyID uint64) (string, error)

	// CreateOutboundCh creates an outbound channel to given address with given initial balance.
	// It takes a context, currency id, to address, initial amount, and the settlement time as arguments.
	// It returns the paych address and error.
	CreateOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int, settlement int64) (string, error)

	// TopupOutboundCh tops up an existing channel with some balance.
	// It takes a context, currency id, to address, top up amount as arguments.
	// It returns the error.
	TopupOutboundCh(ctx context.Context, currencyID uint64, toAddr string, amt *big.Int) error

	// ListOutboundChs lists all the outbound channels (including those are settling.)
	// It takes a context as the argument.
	// It returns a map from currency id to list of paych addresses and error.
	ListOutboundChs(ctx context.Context) (map[uint64][]string, error)

	// ListActiveOutboundChs lists all active outbound channels.
	// It takes a context as the argument.
	// It returns a map from currency id to a list of channel addresses and error.
	ListActiveOutboundChs(ctx context.Context) (map[uint64][]string, error)

	// InspectOutboundCh inspects an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
	// boolean indicates if channel is active, current chain height, settling height and error.
	InspectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error)

	// SettleOutboundCh settles an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	SettleOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// CollectOutboundCh collects an outbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	CollectOutboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// AddInboundCh adds an inbound channel.
	// It takes currency id, channel address and settlement time as arguments.
	// It returns the error.
	AddInboundCh(currencyID uint64, chAddr string, settlement int64) error

	// ListInboundChs lists all inbound channels.
	// It takes a context as the argument.
	// It returns a map from currency id to list of paych addresses and error.
	ListInboundChs(ctx context.Context) (map[uint64][]string, error)

	// InspectInboundCh inspects an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns from address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
	// boolean indicates if channel is active, current chain height, settling height and error.
	InspectInboundCh(ctx context.Context, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error)

	// SettleInboundCh settles an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	SettleInboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// CollectInboundCh collects an inbound payment channel.
	// It takes a context, currency id and channel address as arguments.
	// It returns the error.
	CollectInboundCh(ctx context.Context, currencyID uint64, chAddr string) error

	// ReserveForSelf reserves amount for self.
	// It takes a currency id, to addr, amount as arguments.
	// It returns the error.
	// 给自己预留款项，currency和toaddr确定是预留在哪个out paych上，预留的款项就是amt
	ReserveForSelf(currencyID uint64, toAddr string, amt *big.Int) error

	// ReserveForSelfWithOffer reserves amount for self with given offer.
	// It takes a pay offer as the argument.
	// It returns the error.
	// 给自己预留款项，offer是自己收到的offer。
	// offer的sender和currency id来确定预留在哪个out paych上，预留的款项是通过offer的amt和surcharge去计算
	ReserveForSelfWithOffer(offer *payoffer.PayOffer) error

	// ReserveForOthers reserves amount for others with given offer.
	// It takes a pay offer as the argument.
	// It returns the error.
	// 给别人预留款项，offer是自己发出的offer。
	// 如果offer内没有linked offer，那么根据offer的currency id和recipient去确定预留在哪个out paych上，预留的款项就是offer内的amt。
	// 如果offer内有linked offer，那么根据linked offer的sender和currency id去确定预留在哪个out paych上，预留的款项是通过linked offer的amt和surcharge去计算。
	ReserveForOthers(offer *payoffer.PayOffer) error

	// PayForSelf is used to pay for self.
	// It takes a currency id, to addr, amount as arguments.
	// It returns a voucher, a function to commit, a function to revert and error.
	// 为自己付款，currency id和to addr确定是用哪个out paych，返回的voucher所含的value就是所给的amt
	PayForSelf(currencyID uint64, toAddr string, amt *big.Int) (string, func(), func(), error)

	// PayForSelfWithOffer is used to pay for self with given offer.
	// It takes the amount, a pay offer as arguments.
	// It returns a voucher, a function to commit, a function to revert and error.
	// 为自己付款，offer是自己收到的offer。offer的from addr和currency id来确定用哪个out paych，返回的voucher所含的value就是所给的amt+offer中所写的surcharge
	// 这个offer是我收到的，用reserve给自己的fund
	PayForSelfWithOffer(amt *big.Int, offer *payoffer.PayOffer) (string, func(), func(), error)

	// PayForOthers is used to pay for others with given offer.
	// It takes the amount, a pay offer as arguments.
	// It returns a voucher, a function to commit, a function to revert and error.
	// 为他人付款，offer是自己发出的offer。
	// 如果offer内没有linked offer，那么根据offer的currency id和recipient去确定用哪个out paych，返回的voucher所含的value就是所给的amt。
	// 如果offer内有linked offer，那么根据linked offer的sender和currency id去确定用哪个out paych，返回的voucher所含的value就是amt+linkedoffer中所写的surcharge
	PayForOthers(amt *big.Int, offer *payoffer.PayOffer) (string, func(), func(), error)

	// RecordReceiveForSelf is used to record a receive of payment for self.
	// It takes a currency id, from address, voucher, origin address as arguments.
	// It returns the amount received, a function to commit, a function to revert and error.
	// 记录给自己收钱，通过currency id和from addr去确定收在哪个in paych上，这笔款项将会被记载origin addr上，返回的amt就是voucher中所含的amt
	RecordReceiveForSelf(currencyID uint64, fromAddr string, voucher string, originAddr string) (*big.Int, func(), func(), error)

	// RecordReceiveForOthers is used to record a receive of payment for others.
	// It takes a from address, voucher, pay offer as arguments.
	// It returns the amount received (after surcharge), a function to commit, a function to revert and error.
	// 记录给别人收钱，这里的offer是自己发出去的offer，通过offer的currency id和from addr去确定收在哪个in paych上，这笔款项不会被记载，返回的amt是voucher中所含的amt-offer中所specify的surcharge
	RecordReceiveForOthers(fromAddr string, voucher string, offer *payoffer.PayOffer) (*big.Int, func(), func(), error)

	// Receive will output the amount recorded between last call of this function and this call of the function.
	// It takes a currency id, the origin address as arguments.
	// It returns the amount received and error.
	// 解释：这是一个确认收款的函数，这个函数是真正的收款。它会返回从上次receive到这个receive之间，所有record到的payment。
	Receive(currencyID uint64, originAddr string) (*big.Int, error)
}

/* Notes */
// This manager is complicated, so the notes contains mandarin so I don't forget how to implement.
// 这个manager比较复杂，所以用中文解释给自己，以防止以后忘记。
// Reserve reserves amount to given address for a given offer. The offer can be nil if this reservation
// is for the the node itself or the offer can be an offer signed by this node. Reservation period depends
// on the offer's expiration or if offer is nil, the reservation will be released if no activity happens
// for a period of time.
// 解释：这是一个预留款项的功能。预留款项分为给自己预留，和给别人预留。举两个例子：
// 第一个例子：A 》B
// 这个例子中，点A要从点B处获得一段数据，经过计算需要1000个单位的款项，那么A将首先用这个函数去预留1000的款项给自己，这个时候，因为是给自己预留，所以
// offer可以是nil。offer是nil的时候，一旦超过一定时间，并没有使用pay函数去给自己付款，那么就会被release。
// 第二个例子：A 》B 》C 》D
// 这个例子中，点A首先会层层下去问payment offer，最后一步到点C，点C生成一个C到D的offer，然后为了这个offer会去预留1000单位的钱，然后预留成功的话，就
// 把offer返回给点B，点B这个时候通过计算，算出要发1000单位给D，它需要付1005单位（5是C收的手续费，从C给的offer中可以算出）。B首先生成一个B到D的offer，
// 然后B就会call这个函数去预留为这个offer去预留1005的单位的钱，然后把这个新产生的offer回复给A，A收到了B的offer，算出来如果要发1000单位给D，那么需要
// 准备1010单位的钱（也是B需要收的手续费，这个手续费包含了B付给C的手续费），A这个时候就会call这个函数，但是不会pass offer，给自己预留1010单位的钱。
// 为什么需要这个函数？
// 因为一个channel可以同时给好几个人用，可能你建立这个channel的目的就是去收手续费，这样子的预留机制，可以保证你的一个channel可以给多个人使用，如果没有预留
// 机制，那么可能你的channel的余额只有1000个单位，但是你却发出去了2个1000单位的offer，如果两个offer被同时使用，那么2个offer付到一半就无法继续下去了。
// 要注意，一旦reserved amt在一定时间内没有被touch，那么就会release
//
// Pay pays to address with a given amount for a given offer. There are four possibilites of this offer:
// 1st - offer supplied is nil, which means this pay is for the node itself, it will use the reserved funds for itself.
// 2nd - offer supplied is not nil, and the offer's sender is not the node itself, it will use the reserved funds for
// itself and also based on the offer state to generate amount (amt + surcharge) for this offer.
// 3rd - offer supplied is not nil, and the offer's sender is the node itself, it will use the reserved funds for this
// offer and also it will load the linked offer and based on the linked offer state to generate amount (amt + surcharge)
// for the linked offer.
// 4th - offer supplied is not nil, and the offer's sender is the node itself, it will use the reserved funds for this
// offer but there isn't any linked offer, so it will just pay the specified amount directly (no surchage).
// It takes a currency id, to address, amount, and an offer (can be nil) as arguments.
// It returns a voucher, a function to commit change, a function to revert change and error.
// Note: after payment, this pay can be successful or not depends on the response of the recipient. Commit or Revert function
// must be called after confirmation so this channel can be used by other threads.
// 解释：这是一个去付款的函数。
// 有四种可能性：
// 第一种，就是 A 》B， 这种直接支付的情况，那么offer可以是nil，然后会用reserve给自己的那部分fund。
// 第二种，就是在 A 》B 》C 》D 中 A 的情况，这个时候offer是从B到D的，这样会用reserve给自己的那部分fund，但是会根据offer去计算surchage，然后
// 生成含有surchage的voucher。
// 第三种，就是在 A 》B 》C 》D 中 B 的情况，这个时候offer是从B到D的，这样会用reserve给这个offer的那部分fund，然后会根据offer的linked offer
// 去计算surchage，然后生成含有surcharge的voucher，这个surcharge是从linked offer中去计算出来的。
// 第四种，就是在 A 》B 》C 》D 中 C 的情况，这个时候offer是从C到D的，这样会用reserve给这个offer的那部分fund，然后因为offer没有linked offer，
// 所以不用计算surchage，就会直接生成对应款项的voucher。
// 要注意的是，这个函数会block其他access这个channel的threads。为什么？因为如果同时两个人去生成voucher，可能因为时间上的差异，可能后一个生成的voucher
// 先到了recipient，这样就会导致前一个voucher失效，后一个voucher有多余的payment。这个函数因此会返回两个函数，一个是commit，一个是revert。
// 如果payment是successful的话，那么就需要commit，如果是失败的，那么你可以去revert。commit或者revert，一定要去call，这样其他threads就会被
// unblocked。
// 为什么可以去revert一个payment？其实有另一种办法是你的recipient去签一个refund voucher回来，但是那样会比较复杂，因为每个channel在这个系统里面
// 是有local reputation去控制，而且是长期的关系，如果你收到了一个payment，但是你并没有做到相应的事情，你不应该去记录，因为一旦你记录了，那么之后
// 你就会被block，没有人会来使用你的服务了。同样的道理，你也可以相信你的recipient会去revert，因为一旦不revert，那么你就会block它。加上所有的payment
// 都是micro-payment，如果i你的recipient不revert一笔钱，那也是很少的一笔钱。
//
// RecordReceive record a receive of payment. If offer is nil, meaning that this node is the final recipient of this payment.
// It will record this payment to origin addr. If offer is not nil, meaning that this node is an intermediate recipient of this
// payment. It will use offer to calculate surcharge and return the petty amount (amt in voucher - surcharge), which will be used
// for the proxy payment.
// Note: Commit or Revert function must be called after confirmation so this channel can be used by other threads.
// 解释：这是一个收款的函数。这个函数并不是真正的收款，而是如果这个收款里的目的地是node itself的话，就会记录一笔从origin处收到的钱。
// 有四种可能性：
// 第一种，就是 A 》B， 中 B 的情况，这个时候，from addr是A，origin也是A，offer是nil，有多少钱在voucher里面就会返回多少钱。
// 第二种，就是在 A 》B 》C 》D 中 B 的情况，这个时候，from addr是A，origin是A, offer是从B到D的。这个时候，会根据offer里面的surcharge去计算
// 出实际收到的钱（扣去手续费之后）。
// 第三种，就是在 A 》B 》C 》D 中 C 的情况，这个时候，from addr是B，origin是A，offer是从C到D的。这个时候，会感觉offer里面的surcharge去计算
// 出实际收到的钱（扣去收付费之后）。
// 第四种，就是在 A 》B 》C 》D 中 D 的情况，这个时候，from addr是C，origin是A，offer是nil，有多少钱在voucher里面就会返回多少钱。
// 所以合并起来，如果offer是nil的时候，就会把收到的款项记录给origin，否则的话，收到的款项不会记录给谁。
// 注意，和pay一样，这个也会返回一个commit和revert函数，这个主要是去block其他threads的access。就不会出现我虽然收到了这笔款项。但是因为种种原因，
// 我需要去revert的情况。
