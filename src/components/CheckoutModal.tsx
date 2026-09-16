import React, { useEffect, useRef, useState } from 'react'
import { BookingRecord, MealPlan } from '../types'
import { inr } from '../data/hotels'
import { gstRatePercentFor } from '../lib/pricing'
import { getApiUrl } from '../config/api'
import { getToken } from '../data/auth'

declare global {
  interface Window {
    Razorpay?: new (options: Record<string, unknown>) => { open: () => void; on: (e: string, cb: (r: any) => void) => void }
  }
}

const RAZORPAY_SCRIPT = 'https://checkout.razorpay.com/v1/checkout.js'

/** Loads the Razorpay checkout script once and resolves when it is usable. */
function loadRazorpay(): Promise<boolean> {
  if (window.Razorpay) return Promise.resolve(true)
  return new Promise((resolve) => {
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${RAZORPAY_SCRIPT}"]`)
    if (existing) {
      existing.addEventListener('load', () => resolve(!!window.Razorpay))
      existing.addEventListener('error', () => resolve(false))
      return
    }
    const script = document.createElement('script')
    script.src = RAZORPAY_SCRIPT
    script.async = true
    script.onload = () => resolve(!!window.Razorpay)
    script.onerror = () => resolve(false)
    document.body.appendChild(script)
  })
}


interface CheckoutModalProps {
  propertySlug: string
  propertyName: string
  propertyAddress: string
  roomTypeSlug: string
  roomTypeName: string
  checkIn: string
  checkOut: string
  roomsCount: number
  guestsCount: number
  /** Party split. The server re-derives the extra-bed charge from these. */
  adultsCount: number
  childAges: number[]
  mealPlan?: MealPlan
  totalAmount: number
  onClose: () => void
  onSuccess?: (bookingCode: string) => void
}

export const CheckoutModal: React.FC<CheckoutModalProps> = ({
  propertySlug,
  propertyName,
  propertyAddress,
  roomTypeSlug,
  roomTypeName,
  checkIn,
  checkOut,
  roomsCount,
  guestsCount,
  adultsCount,
  childAges,
  mealPlan,
  totalAmount,
  onClose,
  onSuccess,
}) => {
  const [step, setStep] = useState<'DETAILS' | 'PAYMENT' | 'CONFIRMED'>('DETAILS')
  const [loading, setLoading] = useState(false)
  const [verifying, setVerifying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Stops the confirmation poll from calling setState after the guest closes
  // the modal mid-verification.
  const cancelledRef = useRef(false)
  useEffect(() => () => { cancelledRef.current = true }, [])

  // Guest details state
  const [guestName, setGuestName] = useState('')
  const [guestPhone, setGuestPhone] = useState('')
  const [guestEmail, setGuestEmail] = useState('')
  const [companyName, setCompanyName] = useState('')
  const [gstin, setGstin] = useState('')
  const [showCorporate, setShowCorporate] = useState(false)

  // Created hold / order state
  const [booking, setBooking] = useState<BookingRecord | null>(null)


  // Nights, and the GST slab that follows from the per-room-night rate.
  //
  // The rate itself comes from src/lib/pricing.ts — 5% under ₹7,500 a night,
  // 18% at or above it, per the client on 5 Aug 2026 ("our gst is 5%, so please
  // replace 12% with 5%"). It is not written out here because the same slab has
  // to be applied by the backend when it renders the tax invoice, and a literal
  // in this file is a literal that can drift away from the one on the PDF.
  const nights = Math.max(
    1,
    Math.round((new Date(checkOut).getTime() - new Date(checkIn).getTime()) / (1000 * 60 * 60 * 24))
  )
  // Once the hold exists, the server's figure is the one Razorpay will charge.
  // It can differ from the client-side estimate: the ResAvenue channel manager
  // may have set a rate for these nights that the bundled pricing does not
  // know about, and "Pay ₹X" must be the ₹X the gateway actually takes.
  const payable = booking ? Number(booking.total_amount) : totalAmount
  const ratePerRoomNight = payable / (nights * roomsCount)
  const gstRatePercent = gstRatePercentFor(ratePerRoomNight)
  const taxableBase = Math.round((payable / (1 + gstRatePercent / 100)) * 100) / 100
  const gstAmount = Math.round((payable - taxableBase) * 100) / 100

  const handleInitiateHold = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const res = await fetch(getApiUrl('bookings/initiate'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          // Optional: links the stay to the guest's account when signed in.
          ...(getToken() ? { Authorization: `Bearer ${getToken()}` } : {}),
        },
        body: JSON.stringify({
          propertySlug,
          roomTypeSlug,
          checkIn,
          checkOut,
          roomsCount,
          guestsCount,
          adultsCount,
          childAges,
          guestName: guestName.trim(),
          guestPhone: guestPhone.trim(),
          guestEmail: guestEmail.trim() || undefined,
          companyName: companyName.trim() || undefined,
          gstin: gstin.trim() || undefined,
          mealPlan
        })
      })

      const json = await res.json()
      if (!res.ok || !json.success) {
        throw new Error(json.error || 'Failed to initiate booking hold')
      }

      setBooking(json.data)
      setStep('PAYMENT')
    } catch (err: any) {
      setError(err.message || 'Error communicating with reservation server')
    } finally {
      setLoading(false)
    }
  }

  /**
   * The server is the only authority on whether a booking is paid. Razorpay
   * confirms out-of-band via the webhook, so after checkout closes we poll the
   * booking until the server reports CONFIRMED rather than assuming success.
   */
  const awaitServerConfirmation = async (bookingCode: string): Promise<BookingRecord | null> => {
    const deadline = Date.now() + 90_000
    while (Date.now() < deadline) {
      if (cancelledRef.current) return null
      try {
        const res = await fetch(
          getApiUrl(`bookings/${encodeURIComponent(bookingCode)}?phone=${encodeURIComponent(guestPhone.trim())}`)
        )
        const json = await res.json()
        if (res.ok && json.success && json.data) {
          const record = json.data as BookingRecord
          if (record.booking_status === 'CONFIRMED' && record.payment_status === 'PAID') return record
          if (record.booking_status === 'CANCELLED' || record.booking_status === 'EXPIRED') return record
        }
      } catch {
        // Transient network error — keep polling until the deadline.
      }
      await new Promise((r) => setTimeout(r, 2500))
    }
    return null
  }

  const handlePayNow = async () => {
    if (!booking) return
    setLoading(true)
    setError(null)

    try {
      const orderRes = await fetch(getApiUrl('payments/create-order'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ bookingCode: booking.booking_code }),
      })
      const orderJson = await orderRes.json()
      if (!orderRes.ok || !orderJson.success) {
        throw new Error(orderJson.error || 'Could not start the payment')
      }

      const { orderId, keyId, amount, currency, isSimulated } = orderJson.data

      if (isSimulated || !keyId) {
        // No live Razorpay credentials on this deploy. Do not fabricate a
        // confirmation — the hold is real and expires in 15 minutes.
        throw new Error(
          'Online payment is not enabled on this environment. Your room is held for 15 minutes — ' +
            `call us with booking code ${booking.booking_code} to confirm.`
        )
      }

      const ready = await loadRazorpay()
      if (!ready || !window.Razorpay) {
        throw new Error('Could not reach the payment gateway. Please check your connection and retry.')
      }

      await new Promise<void>((resolve) => {
        const rzp = new window.Razorpay!({
          key: keyId,
          order_id: orderId,
          amount,
          currency: currency || 'INR',
          name: 'Quadis Hotels',
          description: `${propertyName} — ${roomTypeName}`,
          prefill: {
            name: booking.guest_name,
            contact: booking.guest_phone,
            ...(booking.guest_email ? { email: booking.guest_email } : {}),
          },
          notes: { bookingCode: booking.booking_code },
          // Read from the token rather than hardcoding — tokens.css is the only
          // place a colour is allowed to be defined.
          theme: {
            color: getComputedStyle(document.documentElement).getPropertyValue('--gold-deep').trim() || undefined,
          },
          modal: { ondismiss: () => resolve() },
          handler: () => resolve(),
        })
        rzp.on('payment.failed', () => resolve())
        rzp.open()
      })

      setVerifying(true)
      const confirmed = await awaitServerConfirmation(booking.booking_code)
      if (cancelledRef.current) return

      if (confirmed?.booking_status === 'CONFIRMED') {
        setBooking(confirmed)
        setStep('CONFIRMED')
        onSuccess?.(confirmed.booking_code)
      } else if (confirmed?.booking_status === 'CANCELLED' || confirmed?.booking_status === 'EXPIRED') {
        setError('The payment did not go through and the room hold has been released. Please try again.')
      } else {
        setError(
          'We have not received confirmation from the payment gateway yet. If money has left your account, ' +
            `quote booking code ${booking.booking_code} and we will confirm it manually.`
        )
      }
    } catch (err: any) {
      setError(err.message || 'Error communicating with the payment gateway')
    } finally {
      setVerifying(false)
      setLoading(false)
    }
  }

  const handleDownloadInvoice = () => {
    if (!booking) return
    // The server renders the real GST PDF. window.print() produced a blank page
    // because global.css hides #root, which contains the print markup.
    window.open(
      getApiUrl(
        `bookings/${encodeURIComponent(booking.booking_code)}/invoice?phone=${encodeURIComponent(booking.guest_phone)}`
      ),
      '_blank',
      'noopener'
    )
  }

  return (
    <div style={styles.backdrop}>
      <div style={styles.modal}>
        {/* Modal Header */}
        <div style={styles.header}>
          <div>
            <span style={styles.badge}>Quadis Instant Booking & Pay</span>
            <h3 style={styles.title}>{propertyName}</h3>
            <p style={styles.subtitle}>{roomTypeName} • {propertyAddress}</p>
          </div>
          <button onClick={onClose} style={styles.closeBtn} aria-label="Close modal">×</button>
        </div>

        {/* Price Breakdown Banner */}
        <div style={styles.summaryBar}>
          <div style={styles.datesBox}>
            <span style={styles.datesLabel}>Check-In / Out ({nights} Night{nights > 1 ? 's' : ''})</span>
            <strong style={styles.datesText}>{checkIn} → {checkOut}</strong>
            {/* Spell out the party, so a guest paying an extra-bed charge can
                see the headcount the charge came from before they pay it. */}
            <span style={styles.datesLabel}>
              {roomsCount} room{roomsCount > 1 ? 's' : ''} · {adultsCount} adult{adultsCount > 1 ? 's' : ''}
              {childAges.length > 0 && ` · ${childAges.length} child${childAges.length > 1 ? 'ren' : ''}`}
            </span>
          </div>
          <div style={styles.totalBox}>
            <span style={styles.totalLabel}>Total Payable (Incl. {gstRatePercent}% GST)</span>
            <strong style={styles.totalAmount}>{inr(totalAmount)}</strong>
          </div>
        </div>

        {error && <div style={styles.errorAlert}>{error}</div>}

        {/* STEP 1: GUEST & CORPORATE DETAILS */}
        {step === 'DETAILS' && (
          <form onSubmit={handleInitiateHold} style={styles.body}>
            <div style={styles.sectionTitle}>1. Guest Information (For WhatsApp Receipt & Check-in)</div>
            <div style={styles.formGrid}>
              <label style={styles.label}>
                Full Name *
                <input
                  type="text"
                  required
                  style={styles.input}
                  placeholder="e.g. Divyansh Rawat"
                  value={guestName}
                  onChange={(e) => setGuestName(e.target.value)}
                />
              </label>
              <label style={styles.label}>
                WhatsApp Mobile Number *
                <input
                  type="tel"
                  required
                  style={styles.input}
                  placeholder="10-digit mobile (e.g. 9876543210)"
                  value={guestPhone}
                  onChange={(e) => setGuestPhone(e.target.value)}
                />
              </label>
            </div>
            <label style={styles.label}>
              Email Address (Optional — For PDF Invoice copy)
              <input
                type="email"
                style={styles.input}
                placeholder="divyansh@example.com"
                value={guestEmail}
                onChange={(e) => setGuestEmail(e.target.value)}
              />
            </label>

            <div style={styles.toggleCorporate}>
              <label style={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  checked={showCorporate}
                  onChange={(e) => setShowCorporate(e.target.checked)}
                />
                <span>Add Company GSTIN for Corporate Tax Invoice (SAC 996311)</span>
              </label>
            </div>

            {showCorporate && (
              <div style={styles.formGrid}>
                <label style={styles.label}>
                  Company Name
                  <input
                    type="text"
                    style={styles.input}
                    placeholder="Quadis Technologies Pvt Ltd"
                    value={companyName}
                    onChange={(e) => setCompanyName(e.target.value)}
                  />
                </label>
                <label style={styles.label}>
                  Corporate GSTIN
                  <input
                    type="text"
                    style={styles.input}
                    placeholder="09AAACQ1234F1Z9"
                    value={gstin}
                    onChange={(e) => setGstin(e.target.value)}
                  />
                </label>
              </div>
            )}

            {/* GST Slab note */}
            <div style={styles.gstInfo}>
              <span style={{ fontSize: 13, color: 'var(--text-muted-2)' }}>
                ℹ️ SAC Code 996311: Base Tariff ₹{taxableBase.toLocaleString('en-IN')} + {gstRatePercent}% GST (₹{gstAmount.toLocaleString('en-IN')}) = <strong>{inr(totalAmount)}</strong>
              </span>
            </div>

            <button type="submit" disabled={loading} style={styles.ctaBtn}>
              {loading ? 'Securing 15-Min Soft Hold...' : 'Proceed to Payment'}
            </button>
          </form>
        )}

        {/* STEP 2: RAZORPAY & DEMO PAYMENT OPTIONS */}
        {step === 'PAYMENT' && booking && (
          <div style={styles.body}>
            <div style={styles.holdBanner}>
              <strong>15-minute hold active.</strong> Booking Code: <code style={styles.code}>{booking.booking_code}</code>
            </div>

            <div style={styles.sectionTitle}>2. Choose Payment Option</div>
            <p style={{ color: 'var(--text-muted-2)', fontSize: 14, marginBottom: 16 }}>
              You are paying <strong>{inr(payable)}</strong> to confirm room category <strong>{roomTypeName}</strong>.
            </p>

            <div style={styles.paymentActions}>
              <button
                type="button"
                onClick={handlePayNow}
                disabled={loading}
                style={styles.payOnlineBtn}
              >
                {verifying
                  ? 'Confirming payment…'
                  : loading
                    ? 'Opening secure checkout…'
                    : `Pay ${inr(payable)} securely`}
              </button>
              <p style={styles.payNote}>
                {verifying
                  ? 'Do not close this window — we are waiting for the gateway to confirm.'
                  : 'You will be redirected to Razorpay. Your booking is confirmed only once payment succeeds.'}
              </p>
            </div>
          </div>
        )}

        {/* STEP 3: CONFIRMED RECEIPT & INVOICE DOWNLOAD */}
        {step === 'CONFIRMED' && booking && (
          <div style={styles.body}>
            <div style={styles.successHeader}>
              <div style={styles.successIcon}>✓</div>
              <h4 style={styles.successTitle}>Reservation Confirmed & Paid!</h4>
              <p style={styles.successSubtitle}>Booking Code: <strong>{booking.booking_code}</strong></p>
            </div>

            <div style={styles.ticketBox}>
              <div style={styles.ticketRow}><span>Guest Name:</span> <strong>{booking.guest_name}</strong></div>
              <div style={styles.ticketRow}><span>WhatsApp Number:</span> <strong>+91 {booking.guest_phone}</strong></div>
              <div style={styles.ticketRow}><span>Property:</span> <strong>{propertyName}</strong></div>
              <div style={styles.ticketRow}><span>Dates:</span> <strong>{booking.check_in} → {booking.check_out}</strong></div>
              <div style={styles.ticketRow}><span>Payment ID:</span> <strong>{booking.razorpay_payment_id || 'ONLINE_INSTANT'}</strong></div>
              <div style={styles.ticketRow}><span>Amount Paid:</span> <strong>{inr(booking.total_amount)} ({gstRatePercent}% GST)</strong></div>
            </div>

            <div style={styles.whatsAppBadge}>
              WhatsApp confirmation sent to +91 {booking.guest_phone}!
            </div>

            <div style={styles.confirmButtons}>
              <button onClick={handleDownloadInvoice} style={styles.invoiceBtn}>
                Download GST invoice (SAC 996311)
              </button>
              <button onClick={onClose} style={styles.returnBtn}>
                Done & Return
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ... styles remain unchanged ...

const styles: Record<string, React.CSSProperties> = {
  backdrop: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.65)',
    backdropFilter: 'blur(4px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 9999,
    padding: 16,
  },
  modal: {
    backgroundColor: 'var(--surface)',
    borderRadius: 16,
    boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
    width: '100%',
    maxWidth: 600,
    maxHeight: '90vh',
    overflowY: 'auto',
    border: '1px solid var(--border-card)',
  },
  header: {
    padding: '20px 24px',
    borderBottom: '1px solid var(--border-card)',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    backgroundColor: 'var(--bg-cream)',
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
  },
  badge: {
    fontSize: 11,
    fontWeight: 600,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
    backgroundColor: 'var(--warning-bg)',
    color: 'var(--warning)',
    padding: '3px 8px',
    borderRadius: 9999,
    display: 'inline-block',
    marginBottom: 6,
  },
  title: {
    margin: 0,
    fontSize: 20,
    fontWeight: 700,
    color: 'var(--text-primary)',
  },
  subtitle: {
    margin: '4px 0 0 0',
    fontSize: 14,
    color: 'var(--text-muted)',
  },
  closeBtn: {
    background: 'transparent',
    border: 'none',
    fontSize: 24,
    color: 'var(--text-muted)',
    cursor: 'pointer',
    padding: 4,
    lineHeight: 1,
  },
  summaryBar: {
    backgroundColor: 'var(--text-primary)',
    color: 'var(--surface)',
    padding: '16px 24px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: 12,
  },
  datesBox: {
    display: 'flex',
    flexDirection: 'column',
  },
  datesLabel: {
    fontSize: 12,
    color: 'var(--text-muted)',
  },
  datesText: {
    fontSize: 14,
    color: 'var(--surface)',
  },
  totalBox: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-end',
  },
  totalLabel: {
    fontSize: 11,
    color: 'var(--border-card-2)',
  },
  totalAmount: {
    fontSize: 18,
    color: 'var(--bg-warm)',
    fontWeight: 700,
  },
  body: {
    padding: 24,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: 600,
    color: 'var(--text-primary)',
    marginBottom: 16,
  },
  formGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: 16,
    marginBottom: 16,
  },
  label: {
    display: 'flex',
    flexDirection: 'column',
    fontSize: 13,
    fontWeight: 500,
    color: 'var(--text-body)',
    gap: 6,
    marginBottom: 16,
  },
  input: {
    padding: '10px 12px',
    borderRadius: 8,
    border: '1px solid var(--border-card-2)',
    fontSize: 14,
    outline: 'none',
    width: '100%',
    boxSizing: 'border-box',
  },
  toggleCorporate: {
    marginBottom: 16,
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    fontSize: 13,
    fontWeight: 500,
    color: 'var(--gold-deepest)',
    cursor: 'pointer',
  },
  gstInfo: {
    backgroundColor: 'var(--bg-warm)',
    padding: '10px 14px',
    borderRadius: 8,
    marginBottom: 20,
    borderLeft: '4px solid var(--gold)',
  },
  ctaBtn: {
    width: '100%',
    padding: '14px 20px',
    backgroundColor: 'var(--text-primary)',
    color: 'var(--surface)',
    border: 'none',
    borderRadius: 8,
    fontSize: 15,
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'background-color 0.2s',
  },
  holdBanner: {
    backgroundColor: 'var(--success-bg)',
    border: '1px solid var(--success-bg)',
    color: 'var(--success)',
    padding: '12px 16px',
    borderRadius: 8,
    fontSize: 14,
    marginBottom: 20,
  },
  code: {
    fontWeight: 700,
    backgroundColor: 'var(--success-bg)',
    padding: '2px 6px',
    borderRadius: 4,
  },
  paymentActions: {
    display: 'flex',
    flexDirection: 'column',
    gap: 12,
  },
  payOnlineBtn: {
    padding: '16px',
    backgroundColor: 'var(--success)',
    color: 'var(--surface)',
    border: 'none',
    borderRadius: 8,
    fontSize: 16,
    fontWeight: 700,
    cursor: 'pointer',
    boxShadow: '0 4px 6px -1px rgba(5, 150, 105, 0.2)',
  },
  payNote: {
    textAlign: 'center',
    color: 'var(--text-muted)',
    fontSize: 12,
    margin: '4px 0 0 0',
    lineHeight: 1.5,
  },
  successHeader: {
    textAlign: 'center',
    marginBottom: 24,
  },
  successIcon: {
    fontSize: 48,
    marginBottom: 8,
  },
  successTitle: {
    margin: 0,
    fontSize: 22,
    fontWeight: 700,
    color: 'var(--success)',
  },
  successSubtitle: {
    margin: '4px 0 0 0',
    color: 'var(--text-muted-2)',
    fontSize: 15,
  },
  ticketBox: {
    backgroundColor: 'var(--bg-cream)',
    border: '1px solid var(--border-card)',
    borderRadius: 12,
    padding: 16,
    display: 'flex',
    flexDirection: 'column',
    gap: 10,
    marginBottom: 20,
  },
  ticketRow: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: 14,
    color: 'var(--text-body)',
  },
  whatsAppBadge: {
    backgroundColor: 'var(--success-bg)',
    color: 'var(--success)',
    padding: '10px 14px',
    borderRadius: 8,
    fontSize: 13,
    fontWeight: 600,
    textAlign: 'center',
    marginBottom: 24,
  },
  confirmButtons: {
    display: 'flex',
    flexDirection: 'column',
    gap: 12,
  },
  invoiceBtn: {
    padding: '14px',
    backgroundColor: 'var(--gold-deepest)',
    color: 'var(--surface)',
    border: 'none',
    borderRadius: 8,
    fontSize: 15,
    fontWeight: 600,
    cursor: 'pointer',
    textAlign: 'center',
  },
  returnBtn: {
    padding: '12px',
    backgroundColor: 'var(--bg-warm)',
    color: 'var(--text-body)',
    border: 'none',
    borderRadius: 8,
    fontSize: 14,
    fontWeight: 600,
    cursor: 'pointer',
  },
  errorAlert: {
    backgroundColor: 'var(--error-bg)',
    borderBottom: '1px solid var(--error-bg)',
    color: 'var(--error)',
    padding: '12px 24px',
    fontSize: 13,
    fontWeight: 500,
  },
}
