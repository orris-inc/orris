package telegram

import (
	"html"
	"strconv"
)

// EscapeHTML escapes HTML special characters for safe Telegram message formatting
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// Bot message templates (Chinese, HTML format)
// User binding related messages
const (
	// MsgBindMissingCode is shown when user sends /bind without a code
	MsgBindMissingCode = "⚠️ <b>缺少验证码</b>\n\n" +
		"用法：<code>/bind &lt;code&gt;</code>\n\n" +
		"请在网站设置页面获取验证码"

	// MsgBindSuccess is shown when user binding is successful
	MsgBindSuccess = "✅ <b>绑定成功</b>\n\n" +
		"🔔 您将收到以下通知：\n" +
		"  - 订阅到期提醒\n" +
		"  - 流量使用警告\n\n" +
		"使用 /status 查看设置，/unbind 解绑"

	// MsgBindFailed is shown when user binding fails
	MsgBindFailed = "❌ <b>绑定失败</b>\n\n" +
		"验证码无效或已过期\n" +
		"请检查验证码后重试"
)

// User unbind related messages
const (
	// MsgUnbindSuccess is shown when user unbinding is successful
	MsgUnbindSuccess = "✅ <b>已解绑</b>\n\n" +
		"🔕 您将不再收到通知\n\n" +
		"随时使用 /bind &lt;code&gt; 重新连接"

	// MsgUnbindFailed is shown when user unbinding fails
	MsgUnbindFailed = "❌ <b>解绑失败</b>\n\n" +
		"操作失败，请稍后重试"
)

// Status related messages
const (
	// MsgStatusError is shown when getting status fails
	MsgStatusError = "❌ <b>错误</b>\n\n" +
		"获取状态失败，请稍后重试"

	// MsgStatusNotConnected is shown when user is not bound
	MsgStatusNotConnected = "🔗 <b>未连接</b>\n\n" +
		"您的 Telegram 尚未绑定账户\n\n" +
		"<b>绑定步骤：</b>\n" +
		"1️⃣ 访问网站设置页面\n" +
		"2️⃣ 点击「绑定 Telegram」\n" +
		"3️⃣ 复制验证码\n" +
		"4️⃣ 发送 <code>/bind &lt;验证码&gt;</code>"

	// MsgStatusConnectedSimple is shown in polling mode when user is bound (without detailed info)
	MsgStatusConnectedSimple = "📊 <b>已连接</b>\n\n" +
		"您的账户已绑定\n\n" +
		"使用 /unbind 解绑"
)

// Help messages
const (
	// MsgHelpUser is the basic user help message (used in webhook mode)
	MsgHelpUser = "🤖 <b>Orris 通知机器人</b>\n\n" +
		"订阅到期和流量使用提醒服务\n\n" +
		"<b>命令：</b>\n" +
		"  /bind <code>&lt;code&gt;</code> — 绑定账户\n" +
		"  /status — 查看设置\n" +
		"  /unbind — 解绑账户\n" +
		"  /help — 显示帮助\n\n" +
		"<b>开始使用：</b>\n" +
		"在网站设置页面获取验证码，然后发送 <code>/bind &lt;code&gt;</code> 完成绑定"

	// MsgHelpFull is the full help message with admin commands (used in polling mode)
	MsgHelpFull = "🤖 <b>Orris 通知机器人</b>\n\n" +
		"订阅到期和流量使用提醒服务\n\n" +
		"<b>用户命令：</b>\n" +
		"  /bind <code>&lt;code&gt;</code> — 绑定账户\n" +
		"  /status — 查看设置\n" +
		"  /unbind — 解绑账户\n" +
		"  /help — 显示帮助\n\n" +
		"<b>管理员命令：</b>\n" +
		"  /adminbind <code>&lt;code&gt;</code> — 绑定管理员\n\n" +
		"<b>开始使用：</b>\n" +
		"在网站设置页面获取验证码，然后发送 <code>/bind &lt;code&gt;</code> 完成绑定"
)

// Rate limit messages
const (
	// MsgBindRateLimited is shown when user has too many failed attempts
	MsgBindRateLimited = "⚠️ <b>请求过于频繁</b>\n\n" +
		"您的验证尝试次数过多\n" +
		"请15分钟后再试"

	// MsgAdminBindRateLimited is shown when admin has too many failed attempts
	MsgAdminBindRateLimited = "⚠️ <b>请求过于频繁</b>\n\n" +
		"验证尝试次数过多，账户已临时锁定\n" +
		"请30分钟后再试"
)

// Admin binding related messages
const (
	// MsgAdminFeatureNotEnabled is shown when admin service is not configured
	MsgAdminFeatureNotEnabled = "❌ <b>管理员功能未启用</b>\n\n" +
		"请联系系统管理员"

	// MsgAdminFeatureNotEnabledShort is the short version
	MsgAdminFeatureNotEnabledShort = "❌ <b>管理员功能未启用</b>"

	// MsgAdminBindMissingCode is shown when admin sends /adminbind without a code
	MsgAdminBindMissingCode = "⚠️ <b>缺少验证码</b>\n\n" +
		"用法：<code>/adminbind &lt;code&gt;</code>\n\n" +
		"请在管理后台获取验证码"

	// MsgAdminBindMissingCodePolling is the polling mode version (slightly different text)
	MsgAdminBindMissingCodePolling = "⚠️ <b>缺少验证码</b>\n\n" +
		"用法：<code>/adminbind &lt;code&gt;</code>\n\n" +
		"请在管理后台获取验证码"

	// MsgAdminBindFailed is shown when admin binding fails (webhook mode)
	MsgAdminBindFailed = "❌ <b>绑定失败</b>\n\n" +
		"可能原因：\n" +
		"  - 验证码无效或已过期\n" +
		"  - 您不是管理员账户\n" +
		"  - 此 Telegram 已被其他管理员绑定"

	// MsgAdminBindFailedPolling is shown when admin binding fails (polling mode)
	MsgAdminBindFailedPolling = "❌ <b>绑定失败</b>\n\n" +
		"验证码无效、已过期或您不是管理员\n" +
		"请检查后重试"

	// MsgAdminBindSuccess is shown when admin binding is successful (webhook mode)
	MsgAdminBindSuccess = "✅ <b>管理员绑定成功</b>\n\n" +
		"🔔 您将收到以下管理员通知：\n" +
		"  - 节点/代理离线告警\n" +
		"  - 新用户注册通知\n" +
		"  - 支付成功通知\n" +
		"  - 每日/每周业务摘要\n\n" +
		"使用 /adminstatus 查看设置，/adminunbind 解绑"

	// MsgAdminBindSuccessPolling is shown when admin binding is successful (polling mode)
	MsgAdminBindSuccessPolling = "✅ <b>管理员绑定成功</b>\n\n" +
		"🔔 您将收到以下通知：\n" +
		"  - 节点离线告警\n" +
		"  - 新用户注册\n" +
		"  - 支付成功通知\n" +
		"  - 每日/每周报告"

	// MsgAdminUnbindSuccess is shown when admin unbinding is successful
	MsgAdminUnbindSuccess = "✅ <b>管理员已解绑</b>\n\n" +
		"🔕 您将不再收到管理员通知"

	// MsgAdminUnbindFailed is shown when admin unbinding fails
	MsgAdminUnbindFailed = "❌ <b>解绑失败</b>\n\n" +
		"您可能未绑定管理员账户"

	// MsgAdminStatusNotBound is shown when admin is not bound
	MsgAdminStatusNotBound = "🔗 <b>未绑定管理员账户</b>\n\n" +
		"使用 <code>/adminbind &lt;code&gt;</code> 绑定管理员账户"

	// MsgAdminStatusBound is shown when admin is bound
	MsgAdminStatusBound = "📊 <b>管理员通知状态</b>\n\n" +
		"<b>状态：</b> 🟢 已绑定\n\n" +
		"<i>在管理后台修改通知设置</i>"
)

// BuildStatusConnectedMessage builds a detailed connected status message with notification settings
// This is used in webhook mode where we have access to detailed binding info
func BuildStatusConnectedMessage(notifyExpiring bool, expiringDays int, notifyTraffic bool, trafficThreshold int) string {
	return "📊 <b>通知设置</b>\n\n" +
		"<b>状态：</b> 🟢 已连接\n\n" +
		"<b>到期提醒</b>\n" +
		"  " + boolToStatusZH(notifyExpiring) + "\n" +
		"  提前 " + strconv.Itoa(expiringDays) + " 天提醒\n\n" +
		"<b>流量警告</b>\n" +
		"  " + boolToStatusZH(notifyTraffic) + "\n" +
		"  阈值：" + strconv.Itoa(trafficThreshold) + "%\n\n" +
		"<i>在网站修改设置</i>"
}

// boolToStatusZH converts a boolean to a Chinese status string
func boolToStatusZH(b bool) string {
	if b {
		return "✅ 开启"
	}
	return "❌ 关闭"
}

// Callback query related messages
const (
	// MsgCallbackInvalidAction is shown when callback data format is invalid
	MsgCallbackInvalidAction = "❌ 无效操作"

	// MsgCallbackUnknownAction is shown when callback action is not recognized
	MsgCallbackUnknownAction = "❌ 未知操作"

	// MsgCallbackInvalidRequest is shown when callback request is malformed
	MsgCallbackInvalidRequest = "❌ 无效请求"

	// MsgCallbackPermissionDenied is shown when user doesn't have permission
	MsgCallbackPermissionDenied = "❌ 无权限操作"

	// MsgCallbackUnknownResourceType is shown when resource type is not recognized
	MsgCallbackUnknownResourceType = "❌ 未知资源类型"

	// MsgCallbackOperationFailed is shown when operation fails
	MsgCallbackOperationFailed = "❌ 操作失败"

	// MsgCallbackMuteSuccess is the prefix for successful mute (resource name is appended)
	MsgCallbackMuteSuccess = "✅ 已静默此"

	// MsgCallbackUnmuteSuccess is the prefix for successful unmute (resource name is appended)
	MsgCallbackUnmuteSuccess = "✅ 已解除静默"
)
